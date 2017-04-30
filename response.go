package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/zan-xhipe/rest/jsondata"
)

type Response struct {
	Raw     []byte
	display []byte

	resp *http.Response

	Filter       string
	Pretty       bool
	PrettyIndent string

	verbose int
}

func (r *Response) Load(resp *http.Response, s Settings) error {
	r.resp = resp

	r.Pretty = s.Pretty.Bool
	r.PrettyIndent = s.PrettyIndent.String

	switch r.verbose {
	case 1:
		fmt.Println(r.resp.Status)
	case 2, 3:
		// at level 3 display the raw response
		extra := false
		if r.verbose >= 3 {
			extra = true
		}

		dump, err := httputil.DumpResponse(r.resp, extra)
		if err != nil {
			// this is only the verbose logging, so carry on in case of error
			fmt.Println(err)
			break
		}
		fmt.Println(string(dump))
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

	return nil
}

func (r *Response) filter() error {
	data, err := jsondata.New(string(r.Raw))
	if err != nil {
		return err
	}

	out, err := data.Filter(r.Filter)
	if err != nil {
		return err
	}

	if r.Pretty {
		r.display, err = json.MarshalIndent(out, "", r.PrettyIndent)
		if err != nil {
			return err
		}

		// the pretty flag removes quotes from results, this was added for
		// filtered results to make them easier to work with, so you can
		// directly put them into a parameter or header without doing your
		// own trimming. This should be changed if a better UI for this
		// behaviour is figured out.
		r.display = bytes.TrimPrefix(r.display, []byte{'"'})
		r.display = bytes.TrimSuffix(r.display, []byte{'"'})

		return nil
	}

	r.display, err = json.Marshal(out)
	if err != nil {
		return err
	}

	return nil
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
