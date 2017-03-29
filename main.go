package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/boltdb/bolt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "Verbose mode").Short('v').Bool()

	set       = kingpin.Command("set", "initialise rest session")
	setHost   = set.Flag("host", "hostname for the service").String()
	setPort   = set.Flag("port", "port to access the service").Int()
	setScheme = set.Flag("scheme", "scheme used to access the service eg. http, https").String()

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
)

func main() {
	command := kingpin.Parse()
	switch command {
	case "set":
		if err := setValues(); err != nil {
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

func setValues() error {
	if *verbose {
		fmt.Println("opening database", ".rest.db")
	}

	db, err := bolt.Open(".rest.db", 0600, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("service"))
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

		return nil
	})

	return err
}

func getValues() (*url.URL, error) {
	db, err := bolt.Open(".rest.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	u := &url.URL{}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("service"))
		if b == nil {
			return errors.New("no service bucket")
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
