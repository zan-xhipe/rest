package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	configKey   string
	configValue *string
)

func init() {
	config.Arg("key", "specific service setting").StringVar(&configKey)
}

func displayConfig() error {
	return db.View(func(tx *bolt.Tx) error {
		b, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		if configKey != "" {
			displayServiceKey(b, request.Service, configKey)
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

	if asBucket := getBucketFromBucket(b, key); asBucket != nil {
		printBucket(asBucket, 0)
		return
	}
}
