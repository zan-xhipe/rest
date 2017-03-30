package main

import (
	"fmt"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
)

var (
	config      = kingpin.Command("config", "show and alter service configs")
	configKey   *string
	configValue *string
)

func init() {
	config.Arg("service", "service to display").StringVar(&service)
	configKey = config.Arg("key", "specific service setting").String()
	configValue = config.Arg("value", "set the config key to this value").String()

}

func displayConfig() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		switch {
		case service == "":
			printBucket(tx.Bucket([]byte("info")), 0)
			printBucket(tx.Bucket([]byte("services")), 0)
		case *configKey == "":
			printBucket(getBucket(tx, "services."+service), 0)
		case *configValue == "":
			displayServiceKey(tx, service, *configKey)
		default:
			if err := setConfig(tx, service, *configKey, *configValue); err != nil {
				return err
			}
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

func setConfig(tx *bolt.Tx, service, key, value string) error {
	return tx.Bucket([]byte("services")).Bucket([]byte(service)).Put([]byte(key), []byte(value))
}
