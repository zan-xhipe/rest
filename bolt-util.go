package main

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
)

func printBucket(b *bolt.Bucket, level int) {
	padding := strings.Repeat(" ", level*4)
	c := b.Cursor()
	for key, value := c.First(); key != nil; key, value = c.Next() {
		if value == nil {
			nested := b.Bucket(key)
			printBucket(nested, level+1)
		} else {
			fmt.Printf("%s%s: %s\n", padding, key, value)
		}
	}
}

func setString(b *bolt.Bucket, key string, value *string, defaultValue string) error {
	var v string
	switch {
	case value == nil:
		v = defaultValue
	case *value == "":
		v = defaultValue
	default:
		v = *value
	}

	return b.Put([]byte(key), []byte(v))
}

func setInt(b *bolt.Bucket, key string, value *int, defaultValue int) error {
	var v int
	switch {
	case value == nil:
		v = defaultValue
	default:
		v = *value
	}

	buf := make([]byte, 4)
	binary.PutVarint(buf, int64(v))
	return b.Put([]byte(key), buf)
}
