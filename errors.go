package main

import (
	"errors"
	"fmt"
)

type ErrMalformedDB struct {
	Bucket string
}

func (e ErrMalformedDB) Error() string {
	return fmt.Sprintf("malformed database no %s bucket, initialise a service try 'rest help service init' for init help", e.Bucket)
}

type ErrNoService struct {
	Name string
}

func (e ErrNoService) Error() string {
	return fmt.Sprintf("no service %s found", e.Name)
}

type ErrInvalidPath struct {
	Path string
}

func (e ErrInvalidPath) Error() string {
	return fmt.Sprintf("path %s not valid", e.Path)
}

type ErrNoAlias struct {
	Alias string
}

func (e ErrNoAlias) Error() string {
	return fmt.Sprintf("no alias %s defined", e.Alias)
}

var (
	ErrInitDB           = errors.New("no services, run service init")
	ErrNoInfoBucket     = ErrMalformedDB{Bucket: "info"}
	ErrNoServicesBucket = ErrMalformedDB{Bucket: "services"}
	ErrNoPaths          = ErrMalformedDB{Bucket: "paths"}
	ErrNoServiceSet     = errors.New("no service set, use 'rest service use <service>' to set the current service to use")
	ErrNoAliases        = errors.New("no aliases defined")
)
