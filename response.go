package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/boltdb/bolt"
	jmespath "github.com/jmespath/go-jmespath"
)

type Response struct {
	Raw     []byte
	display []byte

	resp *http.Response

	Filter        string
	Pretty        bool
	PrettyIndent  string
	SetParameters map[string]string

	verbose int
}

func (r *Response) Load(resp *http.Response, s Settings) error {
	r.resp = resp

	r.Pretty = s.Pretty.Bool
	r.PrettyIndent = s.PrettyIndent.String
	r.Filter = s.Filter.String
	r.SetParameters = s.SetParameters

	switch r.verbose {
	case 1:
		log.Println(r.resp.Status)
	case 2, 3:
		// at level 3 display the raw response
		extra := false
		if r.verbose >= 3 {
			extra = true
		}

		dump, err := httputil.DumpResponse(r.resp, extra)
		if err != nil {
			// this is only the verbose logging, so carry on in case of error
			log.Println(err)
			break
		}
		log.Println(string(dump))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	r.Raw = body
	if err := r.Prepare(); err != nil {
		return err
	}

	return nil
}

func (r Response) String() string {
	return fmt.Sprint(string(r.display))
}

func (r *Response) Prepare() error {
	switch {
	case r.Filter != "":
		if err := r.filter(); err != nil {
			return err
		}
	case r.Pretty:
		var msg json.RawMessage
		err := json.Unmarshal(r.Raw, &msg)
		if err != nil {
			return err
		}

		r.display, err = json.MarshalIndent(msg, "", r.PrettyIndent)
		if err != nil {
			return err
		}
	default:
		r.display = r.Raw
	}

	if err := r.setParameters(); err != nil {
		return err
	}

	return nil
}

func (r *Response) filter() error {
	var err error
	r.display, err = filter(r.Raw, r.Filter, r.Pretty, r.PrettyIndent)
	if err != nil {
		return err
	}

	return nil
}

func (r *Response) setParameters() error {
	return db.Update(func(tx *bolt.Tx) error {
		current := string(tx.Bucket([]byte("info")).Get([]byte("current")))

		for param, filt := range r.SetParameters {
			result, err := filter(r.Raw, filt, r.Pretty, r.PrettyIndent)
			if err != nil {
				return err
			}

			// get the path for the bucket, this starts with the current service, but ends before the parameter
			// name
			p := strings.Split(param, ".")
			path := strings.Join(append([]string{"services", string(current)}, p[:len(p)-1]...), ".")

			b := getBucket(tx, path)
			if b == nil {
				return ErrInvalidPath{Path: path}
			}

			// the filter returned no result, unset the parameter
			if result == nil {
				unsetBucket(b, fmt.Sprintf("parameters.%s", p[len(p)-1]))
				return nil
			}

			// this needs to be here as each set could be for a different path
			s := NewSettings()
			s.Parameters[p[len(p)-1]] = string(result)

			if err := s.Write(b); err != nil {
				return err
			}
		}

		return nil
	})
}

func filter(data []byte, exp string, pretty bool, indent string) ([]byte, error) {
	var d interface{}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	out, err := jmespath.Search(exp, d)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	if pretty {
		display, err := json.MarshalIndent(out, "", indent)
		if err != nil {
			return nil, err
		}

		// the pretty flag removes quotes from results, this was added for
		// filtered results to make them easier to work with, so you can
		// directly put them into a parameter or header without doing your
		// own trimming. This should be changed if a better UI for this
		// behaviour is figured out.
		display = bytes.TrimPrefix(display, []byte{'"'})
		display = bytes.TrimSuffix(display, []byte{'"'})

		return display, nil
	}

	display, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	return display, nil
}

func (r Response) ExitCode() int {
	status := r.resp.StatusCode
	// exit non zero if not a 200 response
	if status < 200 || status > 300 {
		// if the exit value gets too high it gets mangled
		// so only keep the hundreds
		return status / 100
	}

	return 0
}
