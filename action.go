package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/boltdb/bolt"
)

var (
	alias string
)

func init() {
	// action.Arg("name", "name for the alias action").
	// 	Required().
	// 	StringVar(&alias)

	// action.Arg("method", "the method to use for the action").
	// 	Required().
	// 	StringVar(&request.Method)

	// action.Arg("path", "the path to perform the action on").
	// 	Required().
	// 	StringVar(&request.Path)

	perform.Arg("name", "action to perform").
		Required().
		StringVar(&alias)
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

		a := fmt.Sprintf("%s %s", request.Method, request.Path)

		if err := ab.Put([]byte(alias), []byte(a)); err != nil {
			return err
		}

		return nil
	})
}

func Perform() {
	err := db.View(func(tx *bolt.Tx) error {
		sb, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		b := sb.Bucket([]byte("aliases"))
		if b == nil {
			return ErrNoAliases
		}

		a := string(b.Get([]byte(alias)))
		if a == "" {
			return ErrNoAlias{Alias: alias}
		}

		t := strings.Split(a, " ")
		method := t[0]
		path := t[1]

		request.Method = method
		request.Path = path

		return nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Do(request.Method)
}
