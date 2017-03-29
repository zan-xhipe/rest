package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()

	set        = kingpin.Command("set", "initialise rest session")
	setService = set.Arg("service", "the service to set values for").Required().String()
	setHost    = set.Flag("host", "hostname for the servicpe").String()
	setPort    = set.Flag("port", "port to access the service").Int()
	setScheme  = set.Flag("scheme", "scheme used to access the service eg. http, https").String()

	use          = kingpin.Command("use", "switch service")
	serviceToUse = use.Arg("service", "the service to use").Required().String()

	get     = kingpin.Command("get", "Perform a GET request")
	getPath = get.Arg("url", "url to perform request on").Required().String()

	post     = kingpin.Command("post", "Perform a POST request")
	postPath = post.Arg("url", "url to perform request on").Required().String()
	postData = post.Arg("data", "data to send in the POST request").Required().String()

	put     = kingpin.Command("put", "Perform a PUT request")
	putPath = put.Arg("url", "url to perform request on").Required().String()
	putData = put.Arg("data", "data to send in the PUT request").Required().String()

	delete     = kingpin.Command("delete", "Performa DELETE request")
	deletePath = delete.Arg("url", "url to perform request on").Required().String()

	dbFile string
)

func init() {
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")
}

func main() {
	command := kingpin.Parse()
	switch command {
	case "set":
		if err := setValues(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "use":
		if err := useService(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "get", "post", "put", "delete":
		resp, err := makeRequest(command)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		fmt.Print(string(body))
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

		if b := serviceBucket.Bucket([]byte(*serviceToUse)); b == nil {
			return ErrNoService{Name: *serviceToUse}
		}

		info, err := tx.CreateBucketIfNotExists([]byte("info"))
		if err != nil {
			return err
		}

		err = info.Put([]byte("current"), []byte(*serviceToUse))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func setValues() error {
	if *verbose {
		fmt.Println("opening database", ".rest.db")
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		serviceBucket, err := tx.CreateBucketIfNotExists([]byte("services"))
		if err != nil {
			return err
		}

		b, err := serviceBucket.CreateBucketIfNotExists([]byte(*setService))
		if err != nil {
			return err
		}

		if setScheme != nil {
			if err := b.Put([]byte("scheme"), []byte(*setScheme)); err != nil {
				return err
			}
		}

		if setHost != nil {
			if err := b.Put([]byte("host"), []byte(*setHost)); err != nil {
				return err
			}
		}

		if setPort != nil {
			buf := make([]byte, 4)
			binary.PutVarint(buf, int64(*setPort))
			if err := b.Put([]byte("port"), buf); err != nil {
				return err
			}
		}

		// if this is the first service to be set then set then also make it current service
		if info := tx.Bucket([]byte("info")); info == nil {
			ib, err := tx.CreateBucket([]byte("info"))
			if err != nil {
				return err
			}

			if err := ib.Put([]byte("current"), []byte(*setService)); err != nil {
				return err
			}
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

		u.Scheme = string(b.Get([]byte("scheme")))
		hostname := string(b.Get([]byte("host")))
		port, err := binary.ReadVarint(bytes.NewReader(b.Get([]byte("port"))))
		if err != nil {
			return err
		}
		u.Host = fmt.Sprintf("%s:%d", hostname, port)

		return nil
	})

	return u, err
}

func makeRequest(reqType string) (*http.Response, error) {
	u, err := getValues()
	if err != nil {
		return nil, err
	}

	u.Path = path()
	if *verbose {
		fmt.Println(u)
	}

	client := &http.Client{}

	req, err := http.NewRequest(strings.ToUpper(reqType), u.String(), data())

	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func path() string {
	for _, p := range []*string{getPath, postPath, putPath, deletePath} {
		if *p != "" {
			return *p
		}
	}

	return ""
}

func data() io.Reader {
	for _, d := range []*string{postData, putData} {
		if d != nil {
			return strings.NewReader(*d)
		}
	}

	return nil
}
