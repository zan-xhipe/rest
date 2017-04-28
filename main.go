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
	versionNumber = "0.2"

	verbLevel int

	db     *bolt.DB
	dbFile string

	request Request
	filter  string
)

func init() {
	kingpin.Flag("verbose", "Verbose mode").Short('v').CounterVar(&verbLevel)
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")

	kingpin.Flag("db", "which config database to use").Default(dbFile).StringVar(&dbFile)
}

func main() {
	command := kingpin.Parse()

	var err error
	db, err = bolt.Open(dbFile, 0600, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	switch command {
	case "version":
		fmt.Println(versionNumber)
	case "service init":
		if err := initService(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "service list":
		if err := listServices(); err != nil {
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
	case "service alias":
		if err := addAlias(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "get", "post", "put", "delete":
		Do(command)

	case "perform":
		Perform()
	}
}

func Do(command string) {
	request.Method = command
	resp, err := makeRequest()
	if err != nil {
		fmt.Println("error making request:", err)
		os.Exit(1)
	}

	verbose(1, resp.Status)

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

func verbose(level int, message string) {
	if verbLevel >= level {
		fmt.Println(message)
	}
}

func useService() error {
	err := db.Update(func(tx *bolt.Tx) error {
		serviceBucket := tx.Bucket([]byte("services"))
		if serviceBucket == nil {
			return ErrInitDB
		}

		// Check that the service exists
		if b := serviceBucket.Bucket([]byte(request.Service)); b == nil {
			return ErrNoService{Name: request.Service}
		}

		info := tx.Bucket([]byte("info"))
		if info == nil {
			// If we get here then the db is malformed, examine careully
			// how it happened.
			return ErrNoInfoBucket
		}

		if err := info.Put([]byte("current"), []byte(request.Service)); err != nil {
			return err
		}

		return nil
	})

	return err
}

func makeRequest() (*http.Response, error) {
	// retrieve settings from db
	if err := db.Update(request.LoadSettings); err != nil {
		return nil, err
	}

	req, err := request.Prepare()
	if err != nil {
		return nil, err
	}

	// verbose, verbose logging
	switch verbLevel {
	case 0:
	case 1:
		fmt.Println(request.URL.String())
	case 2:
		dump, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Print(string(dump))
		// this level might nee to be rethought, see if it's actually usefull
	case 3:
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(string(dump))
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
	// verbose, verbose logging
	switch verbLevel {
	case 0:
	case 1:
	case 2:
		dump, err := httputil.DumpResponse(r, false)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(string(dump))
	case 3:
		dump, err := httputil.DumpResponse(r, true)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(string(dump))
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	switch {
	// filtered result
	case filter != "":
		result, err := filterResult(body)
		if err != nil {
			return err
		}
		if err := printJSON(result); err != nil {
			return err
		}
	// pretty result
	case settings.Pretty.Bool:
		var msg json.RawMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			return err
		}
		if err := printJSON(msg); err != nil {
			return err
		}
	// unaltered result
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

// printJSON pretty prints json when it's set
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
	result := string(out)

	// the pretty flag removes quotes from results, this was added for
	// filtered results to make them easier to work with, so you can
	// directly put them into a parameter or header without doing your
	// own trimming. This should be changed if a better UI for this
	// behaviour is figured out.
	if settings.Pretty.Bool {
		result = strings.TrimPrefix(result, "\"")
		result = strings.TrimSuffix(result, "\"")
	}

	fmt.Println(result)
	return nil
}
