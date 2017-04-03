package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

func init() {
	set.Arg("service", "the service to use").Required().StringVar(&service)
	set.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	set.Flag("header", "header to set for each request").StringMapVar(&headers)
	set.Flag("parameter", "set parameter for request").StringMapVar(&parameters)
	set.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	set.Flag("port", "port to access the service").Default("80").IntVar(&port)
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

		if err := setString(b, "scheme", scheme); err != nil {
			return err
		}

		if err := setString(b, "host", host); err != nil {
			return err
		}

		if err := setInt(b, "port", port); err != nil {
			return err
		}

		for header, value := range headers {
			h, err := b.CreateBucketIfNotExists([]byte("headers"))
			if err != nil {
				return err
			}

			if err := h.Put([]byte(header), []byte(value)); err != nil {
				return err
			}
		}

		for param, value := range parameters {
			p, err := b.CreateBucketIfNotExists([]byte("parameters"))
			if err != nil {
				return err
			}

			if err := p.Put([]byte(param), []byte(value)); err != nil {
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
