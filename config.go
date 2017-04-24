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
	configKey = config.Arg("key", "specific service setting").String()
}

func displayConfig() error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		if *configKey != "" {
			displayServiceKey(b, request.Service, *configKey)
		} else {
			fmt.Println(request.Service)
			printBucket(b, 1)
		}

		return nil
	})
}

func displayServiceKey(b *bolt.Bucket, service, key string) {
	if b == nil {
		return
	}
	if v := b.Get([]byte(key)); v == nil {
		printBucket(b.Bucket([]byte(key)), 0)
	} else {
		fmt.Println(string(v))
	}
}
