package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/elgs/gojq"
	homedir "github.com/mitchellh/go-homedir"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	versionNumber = "0.1"

	verbose = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()

	dbFile string

	request   Request
	noHeaders bool
	noQueries bool
	filter    string
)

func init() {

	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")

	kingpin.Flag("db", "which config database to use").Default(dbFile).StringVar(&dbFile)
}

func main() {
	command := kingpin.Parse()

	switch command {
	case "version":
		fmt.Println(versionNumber)
	case "service init":
		if err := initService(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "service set":
		if err := setValue(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "service unset":
		if err := unsetValue(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "service use":
		if err := useService(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "service config":
		if err := displayConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "get", "post", "put", "delete":
		request.Method = command
		resp, err := makeRequest()
		if err != nil {
			fmt.Println("error making request:", err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Println(resp.Status)
		}

		if err := showRequest(resp); err != nil {
			fmt.Println("error displaying result:", err)
			os.Exit(1)
		}

		// exit non zero if not a 200 response
		if resp.StatusCode < 200 || resp.StatusCode > 300 {
			// if the exit value gets too high it gets mangled
			// so only keep the hundreds
			os.Exit(resp.StatusCode / 100)
		}
	}
}

func useService() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		serviceBucket := tx.Bucket([]byte("services"))
		if serviceBucket == nil {
			return ErrNoServicesBucket
		}

		if b := serviceBucket.Bucket([]byte(request.Service)); b == nil {
			return ErrNoService{Name: request.Service}
		}

		info, err := tx.CreateBucketIfNotExists([]byte("info"))
		if err != nil {
			return err
		}

		err = info.Put([]byte("current"), []byte(request.Service))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func makeRequest() (*http.Response, error) {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// retrieve settings from db
	if err := db.Update(request.LoadSettings); err != nil {
		return nil, err
	}

	req, err := request.Prepare()
	if err != nil {
		return nil, err
	}

	if *verbose {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return nil, err
		}

		fmt.Print(string(dump))
	}

	client := &http.Client{}
	return client.Do(req)
}

func usedFlag(b *bool) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) error {
		*b = true
		return nil
	}
}

func showRequest(r *http.Response) error {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	switch {
	case filter != "":
		result, err := filterResult(body)
		if err != nil {
			return err
		}
		if err := printJSON(result); err != nil {
			return err
		}
	case settings.Pretty.Bool:
		var msg json.RawMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return err
		}
		if err := printJSON(msg); err != nil {
			return err
		}
	default:
		fmt.Println(string(body))
	}

	return nil
}

func paramReplacer(parameters map[string]string) *strings.Replacer {
	rep := make([]string, 0, len(parameters))
	for key, value := range parameters {
		rep = append(rep, ":"+key)
		rep = append(rep, value)
	}
	return strings.NewReplacer(rep...)
}

func filterResult(body []byte) (interface{}, error) {
	parser, err := gojq.NewStringQuery(string(body))
	if err != nil {
		return nil, err
	}

	return parser.Query(filter)
}

func printJSON(v interface{}) error {
	var out []byte
	var err error
	if settings.Pretty.Bool {
		out, err = json.MarshalIndent(v, "", settings.PrettyIndent.String)
	} else {
		out, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
