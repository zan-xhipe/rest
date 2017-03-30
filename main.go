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

	set = kingpin.Command("init", "initialise rest session")
	// _          = set.Arg("service", "the service to set values for").Required().Action(setService)
	setHost    = set.Flag("host", "hostname for the servicpe").String()
	setPort    = set.Flag("port", "port to access the service").Int()
	setScheme  = set.Flag("scheme", "scheme used to access the service eg. http, https").String()
	setHeaders = set.Flag("header", "header to set for each request").StringMap()

	use = kingpin.Command("use", "switch service")
	_   = use.Arg("service", "the service to use").Required().Action(setService).String()

	config      = kingpin.Command("config", "show and alter service configs")
	_           = config.Arg("service", "service to display").Action(setService).String()
	configKey   = config.Arg("key", "specific service setting").String()
	configValue = config.Arg("value", "set the config key to this value").String()

	get = kingpin.Command("get", "Perform a GET request")
	_   = get.Arg("path", "url to perform request on").Required().Action(setPath).String()
	_   = get.Flag("service", "the service to use").Action(setService).String()

	post = kingpin.Command("post", "Perform a POST request")
	_    = post.Arg("path", "url to perform request on").Required().Action(setPath).String()
	_    = post.Arg("data", "data to send in the POST request").Required().Action(setData).String()
	_    = post.Flag("service", "the service to use").Action(setService).String()

	put = kingpin.Command("put", "Perform a PUT request")
	_   = put.Arg("path", "url to perform request on").Required().Action(setPath).String()
	_   = put.Arg("data", "data to send in the PUT request").Required().Action(setData).String()
	_   = put.Flag("service", "the service to use").Action(setService).String()

	delete = kingpin.Command("delete", "Performa DELETE request")
	_      = delete.Arg("path", "url to perform request on").Required().Action(setPath).String()
	_      = delete.Flag("service", "the service to use").Action(setService).String()

	dbFile string

	service string
	path    string
	data    io.Reader
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

func displayConfig() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		switch {
		case service == "":
			printBucket(tx.Bucket([]byte("info")), 0)
			printBucket(tx.Bucket([]byte("services")), 0)
		case *configKey == "":
			printBucket(getBucket(tx, "services."+service), 0)
		case *configValue == "":
			displayServiceKey(tx, service, *configKey)
		default:
			if err := setConfig(tx, service, *configKey, *configValue); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func displayServiceKey(tx *bolt.Tx, service, key string) {
	b := getBucket(tx, "services."+service)
	if b == nil {
		return
	}
	if v := b.Get([]byte(key)); v == nil {
		printBucket(b.Bucket([]byte(key)), 0)
	} else {
		fmt.Println(string(v))
	}
}

func setConfig(tx *bolt.Tx, service, key, value string) error {
	return tx.Bucket([]byte("services")).Bucket([]byte(service)).Put([]byte(key), []byte(value))
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

		if err := setString(b, "scheme", setScheme, "http"); err != nil {
			return err
		}

		if err := setString(b, "host", setHost, "localhost"); err != nil {
			return err
		}

		if err := setInt(b, "port", setPort, 80); err != nil {
			return err
		}

		for header, value := range *setHeaders {
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

		u.Scheme = string(b.Get([]byte("scheme")))
		hostname := string(b.Get([]byte("host")))
		port, err := binary.ReadVarint(bytes.NewReader(b.Get([]byte("port"))))
		if err != nil {
			return err
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
	u, headers, err := getValues()
	if err != nil {
		return nil, err
	}

	u.Path = path
	if *verbose {
		fmt.Println(u)
	}

	client := &http.Client{}

	req, err := http.NewRequest(strings.ToUpper(reqType), u.String(), data)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
