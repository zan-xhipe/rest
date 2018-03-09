package main

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"
)

func TestWriteToBucket(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "rest.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	db, err := bolt.Open(tmpfile.Name(), 0600, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte("test"))
		if err != nil {
			return err
		}

		if err := writeString(b, "test-string", sql.NullString{String: "test", Valid: true}); err != nil {
			return err
		}

		if err := writeInt(b, "test-int", sql.NullInt64{Int64: 42, Valid: true}); err != nil {
			return err
		}

		if err := writeBool(b, "test-bool", sql.NullBool{Bool: true, Valid: true}); err != nil {
			return err
		}

		if err := writeDuration(b, "test-duration", NullDuration{Duration: 3 * time.Second, Valid: true}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("test"))
		if b == nil {
			return errors.New("no bucket called test found")
		}

		if v := readString(b, "test-string"); !v.Valid || v.String != "test" {
			return errors.New("test string not read")
		}

		if v := readInt(b, "test-int"); !v.Valid || v.Int64 != 42 {
			return errors.New("test int not read")
		}

		if v := readBool(b, "test-bool"); !v.Valid || !v.Bool {
			return errors.New("test bool not read")
		}

		if v := readDuration(b, "test-duration"); !v.Valid || v.Duration != 3*time.Second {
			return errors.New("test duration not read")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
