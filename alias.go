package main

import (
	"fmt"
	"log"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
)

var (
	aliasDescription string
	aliasParams      map[string]map[string]*string
)

func addAliases(service string) {
	action.Arg("name", "name for the alias action").
		Required().
		StringVar(&request.Alias)

	action.Arg("method", "the method to use for the action").
		StringVar(&request.Method)

	action.Arg("path", "the path to perform the action on").
		StringVar(&request.Path)

	action.Flag("description", "a short description of the alias, will be used in generated help documentation").
		StringVar(&aliasDescription)

	action.Arg("data", "data to be sent with the request").
		StringVar(&request.Data)

	settings.Flags(action, false)

	aliasParams = make(map[string]map[string]*string)

	// parse aliases and make them part of the command, aliases will show up on help,
	// aliases can be called directly as a subcommand of 'rest'
	if err := setAliases(service); err != nil {
		panic(err)
	}

}

func setAliases(current string) error {
	// no aliases without a service
	if current == "" {
		return nil
	}

	if err := db.Open(); err != nil {
		return err
	}
	defer db.Close()

	return db.View(func(tx *bolt.Tx) error {

		services := tx.Bucket([]byte("services"))
		if services == nil {
			return nil
		}

		sb := services.Bucket([]byte(current))
		if sb == nil {
			return nil
		}

		aliases := sb.Bucket([]byte("aliases"))
		if aliases == nil {
			return nil
		}

		return aliases.ForEach(func(k, _ []byte) error {
			b := aliases.Bucket(k)
			desc := fmt.Sprintf(
				"%s\n\n%s",
				string(b.Get([]byte("description"))),
				"All normal request flags are available, but hidden to keep help relevant",
			)

			a := kingpin.Command(string(k), desc)

			// attached data arguments to post and put methods
			method := string(b.Get([]byte("method")))
			if method == "post" || method == "put" {
				a.Arg("data", "data to send in the request").
					StringVar(&request.Data)
			}

			requestFlags(a, true)

			aliasParams[string(k)] = make(map[string]*string)

			// turn path parameters into flags
			path := string(b.Get([]byte("path")))
			for p, _ := range findParams(path) {
				addAliasParam(a, string(k), p)
			}

			// turn header parameters into flags
			if h := b.Bucket([]byte("headers")); h != nil {
				if err := h.ForEach(func(_, value []byte) error {
					for p, _ := range findParams(string(value)) {
						addAliasParam(a, string(k), p)
					}

					return nil
				}); err != nil {
					return err
				}
			}

			// turn query parameters into flags
			if q := b.Bucket([]byte("queries")); q != nil {
				if err := q.ForEach(func(_, value []byte) error {
					for p, _ := range findParams(string(value)) {
						addAliasParam(a, string(k), p)
					}

					return nil
				}); err != nil {
					return err
				}
			}

			data := string(b.Get([]byte("data")))
			for p, _ := range findParams(data) {
				addAliasParam(a, string(k), p)
			}

			return nil
		})
	})
}

func addAliasParam(cmd *kingpin.CmdClause, name, param string) {
	desc := fmt.Sprintf("set :%s parameter", param)
	aliasParams[name][param] = cmd.Flag(param, desc).String()
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

		// complian if the method has not already been set
		switch {
		case a.Get([]byte("method")) == nil && request.Method == "":
			return ErrNoAlias{Alias: request.Alias}
		case request.Method != "":
			if err := a.Put([]byte("method"), []byte(request.Method)); err != nil {
				return err
			}
		}

		switch {
		case a.Get([]byte("path")) == nil && request.Path == "":
			return ErrNoAlias{Alias: request.Alias}
		case request.Method != "":
			if err := a.Put([]byte("path"), []byte(request.Path)); err != nil {
				return err
			}
		}

		if aliasDescription != "" {
			if err := a.Put([]byte("description"), []byte(aliasDescription)); err != nil {
				return err
			}
		}

		if request.Data != "" {
			if err := a.Put([]byte("data"), []byte(request.Data)); err != nil {
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
		data := string(a.Get([]byte("data")))

		request.Method = method
		request.Path = path
		request.Data = data

		// get parameters from alias specific flags
		for param := range aliasParams[name] {
			if *aliasParams[name][param] != "" {
				settings.Parameters[param] = *aliasParams[name][param]
			}
		}

		return nil
	})

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	Do(request.Method)
}
