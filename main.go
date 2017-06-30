package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	versionNumber = "0.6"

	verbLevel int

	db     *bolt.DB
	dbFile string

	request  Request
	response Response
)

func init() {
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")

	kingpin.Version(versionNumber)
	kingpin.Flag("verbose", "Verbose mode").Short('v').CounterVar(&verbLevel)
	kingpin.UsageTemplate(usageTemplate)
	log.SetFlags(0)

	addAliases()
}

func main() {
	command := kingpin.Parse()

	var err error
	db, err = bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	defer luaState.Close()

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

func paramReplacer(parameters map[string]string) *strings.Replacer {
	rep := make([]string, 0, len(parameters))
	for key, value := range parameters {
		rep = append(rep, ":"+key)
		rep = append(rep, value)
	}
	return strings.NewReplacer(rep...)
}
