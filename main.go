package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	versionNumber = "0.5"

	verbLevel int

	db     *bolt.DB
	dbFile string

	request  Request
	response Response
)

func init() {
	kingpin.Flag("verbose", "Verbose mode").Short('v').CounterVar(&verbLevel)
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

	default:
		Perform(command)
	}
}

// Do perform the request, display the response, and exit.
func Do(command string) {
	request.Method = command
	request.verbose = verbLevel

	resp, err := request.Perform()
	if err != nil {
		fmt.Println("error making request:", err)
		os.Exit(1)
	}

	response.verbose = verbLevel
	if err := response.Load(resp, request.Settings); err != nil {
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
