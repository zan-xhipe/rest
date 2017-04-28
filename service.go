package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	settings Settings

	all bool
)

func init() {
	settings = NewSettings()

	set.Arg("path", "only apply settings to this path").StringVar(&request.Path)
	set.Arg("request", "only apply settings when performing specified request type on path").StringVar(&request.Method)

	settings.Flags(set)

	initSrv.Arg("service", "initialise service").Required().StringVar(&request.Service)
	settings.Flags(initSrv)

	unset.Arg("key", "the config key to unset, separate levels with '.'").
		Required().
		StringVar(&configKey)

	use.Arg("service", "the service to use").Required().StringVar(&request.Service)

}

func initService() error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		defaultSettings.Merge(settings)
		settings = defaultSettings

		if err := settings.Write(b); err != nil {
			return err
		}

		// if this is the first service to be set then set then also make it current service
		if info := tx.Bucket([]byte("info")); info == nil {
			ib, err := tx.CreateBucket([]byte("info"))
			if err != nil {
				return err
			}

			if err := ib.Put([]byte("current"), []byte(request.Service)); err != nil {
				return err
			}
		}

		return nil
	})
}

func listServices() error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("services"))
		if b == nil {
			return ErrMalformedDB{Bucket: "services"}
		}

		info := tx.Bucket([]byte("info"))
		if info == nil {
			return ErrMalformedDB{Bucket: "info"}
		}

		current := string(info.Get([]byte("current")))

		return b.ForEach(func(key, _ []byte) error {
			currentIndicator := " "
			if string(key) == current {
				currentIndicator = "*"
			}

			fmt.Printf("%s %s\n", currentIndicator, key)

			return nil
		})
	})
}

func setValue() error {
	var err error
	switch {
	case request.Method != "":
		err = setMethod()
	case request.Path != "":
		err = setPath()
	default:
		err = setService()
	}

	return err
}

func setService() error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		if err := settings.Write(b); err != nil {
			return err
		}

		return nil
	})
}

func setPath() error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.PathBucket(tx)
		if err != nil {
			return err
		}

		if err := settings.Write(b); err != nil {
			return err
		}

		return nil
	})
}

func setMethod() error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.MethodBucket(tx)
		if err != nil {
			return err
		}

		if err := settings.Write(b); err != nil {
			return err
		}

		return nil
	})
}

func unsetValue() error {
	return db.Update(func(tx *bolt.Tx) error {
		sb, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		return unsetBucket(sb, configKey)
	})
}
