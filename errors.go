package main

import (
	"errors"
	"fmt"
)

type ErrMalformedDB struct {
	Bucket string
}

func (e ErrMalformedDB) Error() string {
	return fmt.Sprintf("malformed database no %s bucket", e.Bucket)
}

type ErrNoService struct {
	Name string
}

func (e ErrNoService) Error() string {
	return fmt.Sprintf("no service %s found", e.Name)
}

var (
	ErrNoInfoBucket     = ErrMalformedDB{Bucket: "info"}
	ErrNoServicesBucket = ErrMalformedDB{Bucket: "services"}
	ErrNoServiceSet     = errors.New("no current service set")
)
