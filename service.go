package main

import "github.com/boltdb/bolt"

var (
	settings Settings

	all bool
)

func init() {
	settings = NewSettings()

	set.Arg("service", "the service to use").Required().StringVar(&request.Service)
	set.Arg("path", "only apply settings to this path").StringVar(&request.Path)
	set.Arg("request", "only apply settings when performing specified request type on path").StringVar(&request.Method)

	settings.Flags(set)

	initSrv.Arg("service", "initialise service").Required().StringVar(&request.Service)
	settings.Flags(initSrv)

	unset.Arg("service", "the service to use").Required().StringVar(&request.Service)
	unset.Arg("path", "only apply setting to this path").StringVar(&request.Path)
	unset.Arg("request", "only apply setting when performing specified request type on path").StringVar(&request.Method)
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

	use.Arg("service", "the service to use").Required().StringVar(&request.Service)

}

func initService() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

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

func setValue() error {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	switch {
	case request.Method != "":
		err = setRequestType(db)
	case request.Path != "":
		err = setPath(db)
	default:
		err = setService(db)
	}

	return err
}

func setService(db *bolt.DB) error {
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

func setPath(db *bolt.DB) error {
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

func setRequestType(db *bolt.DB) error {
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
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return err
	}

	switch {
	case request.Method != "":
		err = unsetRequestType(db)
	case request.Path != "":
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
			return b.DeleteBucket([]byte(request.Service))
		}

		return settings.Unset(b)
	})
}

func unsetPath(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.ServiceBucket(tx)
		if err != nil {
			return err
		}

		if all {
			return b.DeleteBucket([]byte(request.Path))
		}

		return settings.Unset(b)
	})
}

func unsetRequestType(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := request.MethodBucket(tx)
		if err != nil {
			return err
		}

		if all {
			return b.Delete([]byte(request.Method))
		}

		return settings.Unset(b)
	})
}
