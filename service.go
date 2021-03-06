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

	settings.Flags(set, false)
	settings.YAMLFlag(set)

	initSrv.Arg("service", "initialise service").Required().StringVar(&request.Service)
	settings.Flags(initSrv, false)
	settings.YAMLFlag(initSrv)

	remSrv.Arg("service", "remove service").Required().StringVar(&request.Service)

	unset.Arg("key", "the config key to unset, separate levels with '.'").
		Required().
		StringVar(&configKey)

	use.Arg("service", "the service to use").
		HintAction(hintServices).
		Required().
		StringVar(&request.Service)

}

func initService() error {
	if yamlFile != nil && *yamlFile != "" {
		// ignore other settings when passing a yaml file
		return WriteYAMLSettings(yamlFile, db, &request)
	}

	// parse flags normally
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.MakeServiceBucket(tx)
		if err != nil {
			return err
		}

		defaultSettings.Merge(settings)
		settings = defaultSettings

		if err := settings.Write(b); err != nil {
			return err
		}

		// if this is the first service to be set then set then also make it current service
		// and initialise the db
		info := tx.Bucket([]byte("info"))
		if info == nil {
			info, err = tx.CreateBucket([]byte("info"))
			if err != nil {
				return err
			}

			if err := info.Put([]byte("version"), []byte(versionNumber)); err != nil {
				return err
			}

		}

		if info.Get([]byte("current")) == nil {
			if err := info.Put([]byte("current"), []byte(request.Service)); err != nil {
				return err
			}
		}

		return nil
	})
}

func removeService() error {
	return db.Update(func(tx *bolt.Tx) error {
		services := tx.Bucket([]byte("services"))
		if services == nil {
			return ErrMalformedDB{Bucket: "services"}
		}

		if err := services.DeleteBucket([]byte(request.Service)); err != nil {
			return err
		}

		info := tx.Bucket([]byte("info"))
		if info == nil {
			return ErrMalformedDB{Bucket: "info"}
		}

		current := string(info.Get([]byte("current")))
		if current == request.Service {
			if err := info.Delete([]byte("current")); err != nil {
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

		current, err := db.CurrentService(tx)
		if err != nil {
			return err
		}

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

func hintServices() []string {
	hints := make([]string, 0)

	if err := db.Open(); err != nil {
		// no hints to provide
		return nil
	}
	defer db.Close()

	// ignore errors as this is just to provide hints, if there are any
	// then just provide no hints
	_ = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("services"))
		if b == nil {
			return ErrMalformedDB{Bucket: "services"}
		}

		return b.ForEach(func(key, _ []byte) error {
			hints = append(hints, string(key))
			return nil
		})
	})

	return hints
}

func setValue() error {
	if yamlFile != nil && *yamlFile != "" {
		// ignore other settings when passing a yaml file
		return WriteYAMLSettings(yamlFile, db, &request)
	}

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
		b, err := request.MakePathBucket(tx)
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
		b, err := request.MakeMethodBucket(tx)
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
