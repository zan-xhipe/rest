package main

import (
	"fmt"
	"os"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"
)

var (
	aliasDescription string
	aliasParams      map[string]map[string]*string
)

func init() {
	action.Arg("name", "name for the alias action").
		Required().
		StringVar(&request.Alias)

	action.Arg("method", "the method to use for the action").
		Required().
		StringVar(&request.Method)

	action.Arg("path", "the path to perform the action on").
		Required().
		StringVar(&request.Path)

	action.Flag("description", "a short description of the alias, will be used in generated help documentation").
		StringVar(&aliasDescription)

	settings.Flags(action)

	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")

	aliasParams = make(map[string]map[string]*string)

	// parse aliases and make them part of the command, aliases will show up on help,
	// aliases can be called directly as a subcommand of 'rest'
	if err := setAliases(); err != nil {
		panic(err)
	}

}

func setAliases() error {
	var err error
	db, err = bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bolt.Tx) error {
		// if the services haven't been initialised there will be nothing here, so all
		// aliases should be ignored and failure to find a particular bucket is safe.
		info := tx.Bucket([]byte("info"))
		if info == nil {
			return nil
		}

		current := info.Get([]byte("current"))
		if current == nil {
			return nil
		}

		services := tx.Bucket([]byte("services"))
		if services == nil {
			return nil
		}

		sb := services.Bucket(current)
		if sb == nil {
			return nil
		}

		aliases := sb.Bucket([]byte("aliases"))
		if aliases == nil {
			return nil
		}

		return aliases.ForEach(func(k, _ []byte) error {
			b := aliases.Bucket(k)
			a := kingpin.Command(string(k), string(b.Get([]byte("description"))))

			// attached data arguments to post and put methods
			method := string(b.Get([]byte("method")))
			if method == "post" || method == "put" {
				a.Arg("data", "data to send in the request").
					StringVar(&request.Data)
			}

			requestFlags(a)

			path := strings.Split(string(b.Get([]byte("path"))), "/")
			aliasParams[string(k)] = make(map[string]*string)
			for _, p := range path {
				if p[0] == ':' {
					param := p[1:]

					desc := fmt.Sprintf("set :%s parameter", param)
					aliasParams[string(k)][param] = a.Flag(param, desc).String()
				}
			}

			return nil
		})
	})
}

func addAlias() error {
	return db.Update(func(tx *bolt.Tx) error {
		sb, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		ab, err := sb.CreateBucketIfNotExists([]byte("aliases"))
		if err != nil {
			return err
		}

		a, err := ab.CreateBucketIfNotExists([]byte(request.Alias))
		if err != nil {
			return err
		}

		if err := a.Put([]byte("method"), []byte(request.Method)); err != nil {
			return err
		}

		if err := a.Put([]byte("path"), []byte(request.Path)); err != nil {
			return err
		}

		if aliasDescription != "" {
			if err := a.Put([]byte("description"), []byte(aliasDescription)); err != nil {
				return err
			}

		}

		if err := settings.Write(a); err != nil {
			return err
		}

		return nil
	})
}

func Perform(name string) {
	request.Alias = name
	err := db.View(func(tx *bolt.Tx) error {
		sb, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		b := sb.Bucket([]byte("aliases"))
		if b == nil {
			return ErrNoAliases
		}

		a := b.Bucket([]byte(name))
		if a == nil {
			return ErrNoAlias{Alias: name}
		}

		method := string(a.Get([]byte("method")))
		path := string(a.Get([]byte("path")))

		request.Method = method
		request.Path = path

		// get parameters from alias specific flags
		for param := range aliasParams[name] {
			if *aliasParams[name][param] != "" {
				settings.Parameters[param] = *aliasParams[name][param]
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Do(request.Method)
}
