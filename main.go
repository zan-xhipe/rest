package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

	set = kingpin.Command("init", "initialise rest session")

	use = kingpin.Command("use", "switch service")

	get = kingpin.Command("get", "Perform a GET request")

	post = kingpin.Command("post", "Perform a POST request")

	put = kingpin.Command("put", "Perform a PUT request")

	delete = kingpin.Command("delete", "Performa DELETE request")

	dbFile string

	scheme     string
	usedScheme bool

	host     string
	usedHost bool

	port     int
	usedPort bool

	service   string
	path      string
	data      string
	noHeaders bool
	headers   map[string]string
)

func init() {
	set.Arg("service", "the service to use").Required().StringVar(&service)
	set.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	set.Flag("header", "header to set for each request").StringMapVar(&headers)
	set.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	set.Flag("port", "port to access the service").Default("80").IntVar(&port)

	use.Arg("service", "the service to use").Required().StringVar(&service)

	get.Arg("path", "url to perform request on").Required().StringVar(&path)
	get.Flag("service", "the service to use").StringVar(&service)
	get.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	get.Flag("header", "set header for request").StringMapVar(&headers)
	get.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	get.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	get.Flag("port", "port to access the service").Default("80").Action(usedFlag(&usedPort)).IntVar(&port)

	post.Arg("path", "url to perform request on").Required().StringVar(&path)
	post.Arg("data", "data to send in the request").Required().StringVar(&data)
	post.Flag("service", "the service to use").StringVar(&service)
	post.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	post.Flag("header", "set header for request").StringMapVar(&headers)
	post.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	post.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	post.Flag("port", "port to access the service").Default("80").Action(usedFlag(&usedPort)).IntVar(&port)

	put.Arg("path", "url to perform request on").Required().StringVar(&path)
	put.Arg("data", "data to send in the request").Required().StringVar(&data)
	put.Flag("service", "the service to use").StringVar(&data)
	put.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	put.Flag("header", "set header for request").StringMapVar(&headers)
	put.Flag("scheme", "scheme used to access the service").Default("http").StringVar(&scheme)
	put.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	put.Flag("port", "port to access the service").Default("80").Action(usedFlag(&usedPort)).IntVar(&port)

	delete.Arg("path", "url to perform request on").Required().StringVar(&path)
	delete.Flag("service", "the service to use").StringVar(&path)
	delete.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	delete.Flag("header", "set header for request").StringMapVar(&headers)
	delete.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	delete.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	delete.Flag("port", "port to access the service").Default("80").Action(usedFlag(&usedPort)).IntVar(&port)

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
			log.Println(err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
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

		b, err := serviceBucket.CreateBucketIfNotExists([]byte(service))
		if err != nil {
			return err
		}

		if err := setString(b, "scheme", scheme); err != nil {
			return err
		}

		if err := setString(b, "host", host); err != nil {
			return err
		}

		if err := setInt(b, "port", port); err != nil {
			return err
		}

		for header, value := range headers {
			h, err := b.CreateBucketIfNotExists([]byte("headers"))
			if err != nil {
				return err
			}

			if err := h.Put([]byte(header), []byte(value)); err != nil {
				return err
			}

		}

		// if this is the first service to be set then set then also make it current service
		if info := tx.Bucket([]byte("info")); info == nil {
			ib, err := tx.CreateBucket([]byte("info"))
			if err != nil {
				return err
			}

			if err := ib.Put([]byte("current"), []byte(service)); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func getValues() (*url.URL, map[string]string, error) {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	u := &url.URL{}
	headers := make(map[string]string)
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

		if h := b.Bucket([]byte("headers")); h != nil {
			c := h.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				headers[string(k)] = string(v)
			}
		}

		return nil
	})

	return u, headers, err
}

func makeRequest(reqType string) (*http.Response, error) {
	u, h, err := getValues()
	if err != nil {
		return nil, err
	}

	u.Path = path
	if *verbose {
		fmt.Println(u)
	}

	client := &http.Client{}

	req, err := http.NewRequest(strings.ToUpper(reqType), u.String(), strings.NewReader(data))

	if !noHeaders {
		for key, value := range headers {
			h[key] = value
		}

		for key, value := range h {
			req.Header.Set(key, value)
		}
	}

	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func usedFlag(b *bool) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) error {
		*b = true
		return nil
	}
}
