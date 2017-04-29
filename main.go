package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	versionNumber = "0.2"

	verbLevel int

	db     *bolt.DB
	dbFile string

	request  Request
	response Response
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
	case "service remove":
		if err := removeService(); err != nil {
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

// Do perform the request, display the response, and exit.
func Do(command string) {
	request.Method = command
	resp, err := makeRequest()
	if err != nil {
		fmt.Println("error making request:", err)
		os.Exit(1)
	}

	response.verbose = verbLevel
	if err := response.Load(resp, settings); err != nil {
		fmt.Println("error displaying result:", err)
		os.Exit(1)
	}

	fmt.Println(response)

	os.Exit(response.ExitCode())
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

func paramReplacer(parameters map[string]string) *strings.Replacer {
	rep := make([]string, 0, len(parameters))
	for key, value := range parameters {
		rep = append(rep, ":"+key)
		rep = append(rep, value)
	}
	return strings.NewReplacer(rep...)
}
