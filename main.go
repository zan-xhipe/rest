package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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

	service     string
	requestPath string
	data        string
	noHeaders   bool
	filter      string
)

func init() {
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
		if err := runInit(); err != nil {
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

func getValues() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b, err := getServiceBucket(tx)
		if err != nil {
			return err
		}

		pb := getDefinedPath(b)
		if pb != nil {
			if err := getBool(pb, "override", &override); err != nil {
				return err
			}
		}

		stored := Settings{}

		switch {
		case override:
			stored.Read(pb)
			settings = stored.Merge(settings)
		case pb != nil:
			stored.Read(b)
			pathStored := Settings{}
			pathStored.Read(pb)
			settings = stored.Merge(pathStored).Merge(settings)
		default:
			stored.Read(b)
			settings = stored.Merge(settings)
		}
		return nil
	})

	return err
}

func getServiceBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	info := tx.Bucket([]byte("info"))
	if info == nil {
		return nil, ErrNoInfoBucket
	}

	current := info.Get([]byte("current"))
	if current == nil {
		return nil, ErrNoServiceSet
	}

	serviceBucket := tx.Bucket([]byte("services"))
	if serviceBucket == nil {
		return nil, ErrNoServicesBucket
	}

	b := serviceBucket.Bucket(current)
	if b == nil {
		return nil, ErrNoService{Name: string(current)}
	}

	return b, nil
}

func getDefinedPath(b *bolt.Bucket) *bolt.Bucket {
	pb := b.Bucket([]byte("paths"))
	if pb == nil {
		return nil
	}

	c := pb.Cursor()
	for key, _ := c.First(); key != nil; key, _ = c.Next() {
		// TODO: improve matching
		if string(key) == requestPath {
			return pb.Bucket(key)
		}
	}

	return nil
}

func makeRequest(reqType string) (*http.Response, error) {
	err := getValues()
	if err != nil {
		return nil, err
	}

	u := settings.URL()

	params := paramReplacer(settings.Parameters)
	u.Path = path.Join(settings.BasePath.String, params.Replace(requestPath))
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
		for key, value := range settings.Headers {
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
