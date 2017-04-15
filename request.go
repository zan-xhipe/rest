package main

import "github.com/boltdb/bolt"

type Request struct {
	Service string
	Method  string
	Path    string
	Data    string
}

func (r Request) ServiceBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	if r.Service == "" {
		info := tx.Bucket([]byte("info"))
		current := info.Get([]byte("current"))
		r.Service = string(current)
	}

	sb, err := tx.CreateBucketIfNotExists([]byte("services"))
	if err != nil {
		return nil, err
	}

	b, err := sb.CreateBucketIfNotExists([]byte(r.Service))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (r Request) PathBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	s, err := r.ServiceBucket(tx)
	if err != nil {
		return nil, err
	}

	pb, err := s.CreateBucketIfNotExists([]byte("paths"))
	if err != nil {
		return nil, err
	}

	b, err := pb.CreateBucketIfNotExists([]byte(r.Path))
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (r Request) MethodBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	s, err := r.PathBucket(tx)
	if err != nil {
		return nil, err
	}

	b, err := s.CreateBucketIfNotExists([]byte(r.Method))
	if err != nil {
		return nil, err
	}

	return b, nil
}
