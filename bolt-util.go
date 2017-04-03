package main

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
)

func getBucket(tx *bolt.Tx, path string) *bolt.Bucket {
	p := strings.Split(path, ".")
	if len(p) < 1 {
		return nil
	}

	b := tx.Bucket([]byte(p[0]))
	if b == nil {
		return nil
	}

	for _, v := range p[1:] {
		b = b.Bucket([]byte(v))
		if b == nil {
			return nil
		}
	}

	return b
}

func printBucket(b *bolt.Bucket, level int) {
	padding := strings.Repeat(" ", level*4)
	c := b.Cursor()
	for key, value := c.First(); key != nil; key, value = c.Next() {
		if value == nil {
			nested := b.Bucket(key)
			fmt.Printf("%s%s:\n", padding, string(key))
			printBucket(nested, level+1)
		} else {
			fmt.Printf("%s%s: %s\n", padding, key, value)
		}
	}
}

func bucketMap(b *bolt.Bucket, m *map[string]string) {
	temp := make(map[string]string)
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		temp[string(k)] = string(v)
	}
	for key, value := range *m {
		temp[key] = value
	}
	*m = temp
}

func setString(b *bolt.Bucket, key, value string) error {
	return b.Put([]byte(key), []byte(value))
}

func setInt(b *bolt.Bucket, key string, value int) error {
	buf := make([]byte, 4)
	binary.PutVarint(buf, int64(value))
	return b.Put([]byte(key), buf)
}
