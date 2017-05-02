package main

import (
	"fmt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"
)

func init() {
	action.Arg("name", "name for the alias action").
		Required().
		StringVar(&alias)

	action.Arg("method", "the method to use for the action").
		Required().
		StringVar(&request.Method)

	action.Arg("path", "the path to perform the action on").
		Required().
		StringVar(&request.Path)

	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	dbFile = fmt.Sprintf("%s/%s", dir, ".rest.db")
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

		return aliases.ForEach(setAlias)
	})
}

func setAlias(key, value []byte) error {
	a := kingpin.Command(string(key), "")

	a.Arg("data", "data to send in the request").
		StringVar(&request.Data)
	requestFlags(a)

	return nil
}
