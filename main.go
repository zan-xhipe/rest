package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	versionNumber = "0.9"

	verbLevel int

	db = &DB{}

	request  Request
	response Response

	service string
)

func init() {
	kingpin.Version(versionNumber)
	kingpin.Flag("verbose", "Verbose mode").Short('v').CounterVar(&verbLevel)
	kingpin.Flag("service", "The service to use").StringVar(&request.Service)
	kingpin.UsageTemplate(usageTemplate)
	log.SetFlags(0)

	// we have to parse the service flag twice so that we can change which aliases
	// are loaded in when parsing the full command
	serviceSelection := flag.NewFlagSet("", flag.ContinueOnError)
	serviceSelection.StringVar(&request.Service, "service", "", "Which service to use for the current command")
	// We ignore errors here as we are only trying to parse one flag, we only care about parse errors from kingpin
	_ = serviceSelection.Parse(os.Args[1:])

	addAliases(request.Service)
}

func main() {
	command := kingpin.Parse()

	if verbLevel > 1 && request.Service != "" {
		log.Println("using ", request.Service)
	}

	if err := db.Open(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	if luaState != nil {
		defer luaState.Close()
	}

	switch command {
	case "version":
		displayVersion()
	case "service init":
		if err := initService(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service remove":
		if err := removeService(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service list":
		if err := listServices(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service set":
		if err := setValue(); err != nil {
			log.Println(err)
			os.Exit(1)
		}

	case "service unset":
		if err := unsetValue(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service use":
		if err := useService(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service config":
		if err := displayConfig(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
	case "service alias":
		if err := addAlias(); err != nil {
			log.Println(err)
			os.Exit(1)
		}

	case "get", "post", "put", "delete", "patch", "options", "head":
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
		log.Println("error making request:", err)
		if resp != nil {
			log.Println(resp.Body)
			resp.Body.Close()
		}
		os.Exit(1)
	}

	response.verbose = verbLevel
	if err := response.Load(resp, request.Settings); err != nil {
		log.Println("error displaying result:", err)
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

func displayVersion() {
	var dbVersion string
	noDB := errors.New("no db")
	db.View(func(tx *bolt.Tx) error {
		info := tx.Bucket([]byte("info"))
		if info == nil {
			return noDB
		}

		v := info.Get([]byte("version"))
		if v == nil {
			return noDB
		}

		dbVersion = string(v)
		return nil
	})

	if dbVersion == "" || versionNumber == dbVersion {
		fmt.Println(versionNumber)
	} else {
		fmt.Printf("%s, db: %s, version don't match, the database will have to be updated", versionNumber, dbVersion)
	}

}

func usedFlag(b *bool) func(*kingpin.ParseContext) error {
	return func(*kingpin.ParseContext) error {
		*b = true
		return nil
	}
}
