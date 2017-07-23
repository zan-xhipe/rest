package main

import (
	"fmt"

	"github.com/boltdb/bolt"
	homedir "github.com/mitchellh/go-homedir"
)

type DB struct {
	*bolt.DB
	filename string
}

func (db *DB) Open() error {
	dir, err := homedir.Dir()
	if err != nil {
		return err
	}
	db.filename = fmt.Sprintf("%s/%s", dir, ".rest.db")

	db.DB, err = bolt.Open(db.filename, 0600, nil)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) CurrentService(tx *bolt.Tx) (string, error) {
	info := tx.Bucket([]byte("info"))
	if info == nil {
		return "", ErrMalformedDB{Bucket: "info"}
	}

	return string(info.Get([]byte("current"))), nil
}
