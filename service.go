package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

var (
	settings Settings

	requestType string
	settingKey  string

	all bool
)

func init() {
	settings = NewSettings()

	set.Arg("service", "the service to use").Required().StringVar(&service)
	set.Arg("path", "only apply settings to this path").StringVar(&requestPath)
	set.Arg("request", "only apply settings when performing specified request type on path").StringVar(&requestType)

	settings.Flags(set)

	initSrv.Arg("service", "initialise service").Required().StringVar(&service)
	settings.Flags(initSrv)

	unset.Arg("service", "the service to use").Required().StringVar(&service)
	unset.Arg("path", "only apply setting to this path").StringVar(&requestPath)
	unset.Arg("request", "only apply setting when performing specified request type on path").StringVar(&requestType)
	unset.Flag("all", "delete entire config bucket").BoolVar(&all)
	unset.Flag("scheme", "unset scheme").BoolVar(&settings.Scheme.Valid)
	unset.Flag("host", "unset host").BoolVar(&settings.Host.Valid)
	unset.Flag("port", "unset port").BoolVar(&settings.Port.Valid)
	unset.Flag("base-path", "unset base path").BoolVar(&settings.BasePath.Valid)
	unset.Flag("header", "unset headers").StringMapVar(&settings.Headers)
	unset.Flag("parameter", "unset parameters").StringMapVar(&settings.Parameters)
	unset.Flag("query", "unset query parameters").StringMapVar(&settings.Queries)
	unset.Flag("pretty", "unset pretty").BoolVar(&settings.Pretty.Valid)
	unset.Flag("pretty-indent", "unset Pretty indent").BoolVar(&settings.PrettyIndent.Valid)

	use.Arg("service", "the service to use").Required().StringVar(&service)

}

func initService() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		serviceBucket, err := tx.CreateBucketIfNotExists([]byte("services"))
		if err != nil {
			return err
		}

		b, err := serviceBucket.CreateBucketIfNotExists([]byte(service))
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

			if err := ib.Put([]byte("current"), []byte(service)); err != nil {
				return err
			}
		}

		return nil
	})
}

func setValue() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	switch {
	case requestType != "":
		err = setRequestType(db)
	case requestPath != "":
		err = setPath(db)
	default:
		err = setService(db)
	}

	return err
}

func setService(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, fmt.Sprintf("services.%s", service))

		if err := settings.Write(b); err != nil {
			return err
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

		return nil
	})
}

func setRequestType(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		pb := getBucket(tx, fmt.Sprintf("services.%s.paths.%s", service, requestPath))
		if pb == nil {
			return ErrMalformedDB{Bucket: requestPath}
		}
		b, err := pb.CreateBucketIfNotExists([]byte(requestType))
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
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}

	switch {
	case requestType != "":
		err = unsetRequestType(db)
	case requestPath != "":
		err = unsetPath(db)
	default:
		err = unsetService(db)
	}

	return err
}

func unsetService(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, "services")
		if all {
			return b.DeleteBucket([]byte(service))
		}

		return settings.Unset(b)
	})
}

func unsetPath(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, fmt.Sprintf("services.%s", service))
		if all {
			return b.DeleteBucket([]byte(requestPath))
		}

		return settings.Unset(b)
	})
}

func unsetRequestType(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, fmt.Sprintf("services.%s.%s", service, requestPath))
		if all {
			return b.Delete([]byte(requestType))
		}

		return settings.Unset(b)
	})
}
