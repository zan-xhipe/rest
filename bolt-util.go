package main

import (
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
