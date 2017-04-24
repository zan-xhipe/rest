package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	configKey   *string
	configValue *string
)

func init() {
	config.Arg("service", "service to display").StringVar(&request.Service)
	configKey = config.Arg("key", "specific service setting").String()
}

func displayConfig() error {
	err := db.Update(func(tx *bolt.Tx) error {
		switch {
		case request.Service == "":
			printBucket(tx.Bucket([]byte("info")), 0)
			printBucket(tx.Bucket([]byte("services")), 0)
		case *configKey == "":
			printBucket(getBucket(tx, "services."+request.Service), 0)
		default:
			displayServiceKey(tx, request.Service, *configKey)
		}

		return nil
	})

	return err
}

func displayServiceKey(tx *bolt.Tx, service, key string) {
	b := getBucket(tx, "services."+service)
	if b == nil {
		return
	}
	if v := b.Get([]byte(key)); v == nil {
		printBucket(b.Bucket([]byte(key)), 0)
	} else {
		fmt.Println(string(v))
	}
}
