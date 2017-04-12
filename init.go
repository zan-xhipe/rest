package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	override bool
	settings Settings
)

func init() {
	set.Arg("service", "the service to use").Required().StringVar(&service)
	set.Arg("path", "only apply settings to this path").StringVar(&requestPath)

	settings.Flags(set)

	set.Flag("override-default", "if set then path specific settings completely override service level settings, otherwise the default behaviour is to merge the path settings with base settings, with path settings taking precedent.  This is only valid if a path is specified").BoolVar(&override)
}

func runInit() error {
	if *verbose {
		fmt.Println("opening database", ".rest.db")
	}

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	if requestPath == "" {
		setService(db)
	} else {
		setPath(db)
	}

	return err
}

func setService(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		serviceBucket, err := tx.CreateBucketIfNotExists([]byte("services"))
		if err != nil {
			return err
		}

		b, err := serviceBucket.CreateBucketIfNotExists([]byte(service))
		if err != nil {
			return err
		}

		// ensure that default settings get written
		settings = defaultSettings.Merge(settings)

		if err := settings.Write(b); err != nil {
			return err
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
}

func setPath(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		sb := getBucket(tx, fmt.Sprintf("services.%s", service))
		pb, err := sb.CreateBucketIfNotExists([]byte("paths"))
		if err != nil {
			return err
		}

		b, err := pb.CreateBucketIfNotExists([]byte(requestPath))
		if err != nil {
			return err
		}

		if err := settings.Write(b); err != nil {
			return err
		}

		if err := setBool(b, "override", override); err != nil {
			return err
		}

		return nil
	})
}
