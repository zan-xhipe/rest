package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

	scheme     string
	usedScheme bool

	host     string
	usedHost bool

	port     int
	usedPort bool

	service    string
	path       string
	data       string
	noHeaders  bool
	headers    map[string]string
	filter     string
	parameters map[string]string

	pretty           bool
	prettyIndent     string
	usedPrettyIndent bool
)

func init() {
	headers = make(map[string]string)
	parameters = make(map[string]string)

	use.Arg("service", "the service to use").Required().StringVar(&service)

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
	case "init":
		if err := setValues(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "use":
		if err := useService(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "config":
		if err := displayConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	case "get", "post", "put", "delete":
		resp, err := makeRequest(command)
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

		if b := serviceBucket.Bucket([]byte(service)); b == nil {
			return ErrNoService{Name: service}
		}

		info, err := tx.CreateBucketIfNotExists([]byte("info"))
		if err != nil {
			return err
		}

		err = info.Put([]byte("current"), []byte(service))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func getValues() (*url.URL, error) {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	u := &url.URL{}
	err = db.View(func(tx *bolt.Tx) error {
		info := tx.Bucket([]byte("info"))
		if info == nil {
			return ErrNoInfoBucket
		}

		current := info.Get([]byte("current"))
		if current == nil {
			return ErrNoServiceSet
		}

		serviceBucket := tx.Bucket([]byte("services"))
		if serviceBucket == nil {
			return ErrNoServicesBucket
		}

		b := serviceBucket.Bucket(current)
		if b == nil {
			return ErrNoService{Name: string(current)}
		}

		if usedScheme {
			u.Scheme = scheme
		} else {
			u.Scheme = string(b.Get([]byte("scheme")))

		}

		hostname := host
		if !usedHost {
			hostname = string(b.Get([]byte("host")))
		}

		if !usedPort {
			p, err := binary.ReadVarint(bytes.NewReader(b.Get([]byte("port"))))
			if err != nil {
				return err
			}
			port = int(p)
		}

		u.Host = fmt.Sprintf("%s:%d", hostname, port)

		bucketMap(b.Bucket([]byte("headers")), &headers)
		bucketMap(b.Bucket([]byte("parameters")), &parameters)

		if err := getBool(b, "pretty", &pretty); err != nil {
			return err
		}

		if pretty && !usedPrettyIndent {
			prettyIndent = string(b.Get([]byte("pretty-indent")))
		}

		return nil
	})

	return u, err
}

func makeRequest(reqType string) (*http.Response, error) {
	u, err := getValues()
	if err != nil {
		return nil, err
	}

	params := paramReplacer(parameters)
	u.Path = params.Replace(path)
	data = params.Replace(data)
	if *verbose {
		fmt.Println(u)
	}

	client := &http.Client{}

	req, err := http.NewRequest(strings.ToUpper(reqType), u.String(), strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	if !noHeaders {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

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
	case pretty:
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
	if pretty {
		out, err = json.MarshalIndent(v, "", prettyIndent)
	} else {
		out, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
