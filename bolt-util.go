package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
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

func getBucketFromBucket(bucket *bolt.Bucket, path string) *bolt.Bucket {
	p := strings.Split(path, ".")
	if len(p) < 1 {
		return nil
	}

	b := bucket.Bucket([]byte(p[0]))
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

func unsetBucket(b *bolt.Bucket, key string) error {
	p := strings.Split(key, ".")
	if len(p) < 1 {
		return nil
	}

	for i := range p {
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if string(k) == p[i] {
				// leaf
				if v != nil {
					return b.Delete(k)
				}

				// we are deleting the bucket
				if i == len(p)-1 {
					return b.DeleteBucket(k)
				}

				// go deeper
				b = b.Bucket(k)
				break
			}
		}
	}

	return nil
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
	if b == nil {
		return
	}

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

func setBool(b *bolt.Bucket, key string, value bool) error {
	return b.Put([]byte(key), []byte(fmt.Sprint(value)))
}

func getBool(b *bolt.Bucket, key string, value *bool) error {
	if v := b.Get([]byte(key)); v != nil {
		p, err := strconv.ParseBool(string(v))
		if err != nil {
			return err
		}
		if !*value {
			*value = p
		}
	}

	return nil
}
