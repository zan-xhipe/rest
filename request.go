package main

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/boltdb/bolt"
)

type Request struct {
	Service string
	Method  string
	Path    string
	Data    string

	Settings  Settings
	NoQueries bool
	NoHeaders bool

	URL url.URL
}

func (r Request) Prepare() (*http.Request, error) {
	// prepare the url
	r.URL = r.Settings.URL()
	params := paramReplacer(r.Settings.Parameters)

	r.URL.Path = path.Join(r.Settings.BasePath.String, params.Replace(r.Path))
	r.Data = params.Replace(r.Data)

	if !r.NoQueries {
		q := r.URL.Query()
		for key, value := range r.Settings.Queries {
			q.Set(key, value)
		}
		r.URL.RawQuery = q.Encode()
	}

	// prepare the request
	req, err := http.NewRequest(
		strings.ToUpper(r.Method),
		r.URL.String(),
		strings.NewReader(r.Data),
	)
	if err != nil {
		return nil, err
	}

	if !r.NoHeaders {
		for key, value := range r.Settings.Headers {
			req.Header.Set(key, value)
		}
	}

	return req, nil
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

func (r *Request) LoadSettings(tx *bolt.Tx) error {
	sb, pb, mb, err := request.Match(tx)
	if err != nil {
		return err
	}

	r.Settings = NewSettings()

	// no service based settings
	if sb == nil {
		r.Settings.Merge(settings)
		return nil
	}

	// load service settings as base
	r.Settings = LoadSettings(sb)

	if pb == nil {
		r.Settings.Merge(settings)
		return nil
	}

	r.Settings.Merge(LoadSettings(pb))

	if mb == nil {
		r.Settings.Merge(settings)
		return nil
	}

	r.Settings.Merge(LoadSettings(mb))
	r.Settings.Merge(settings)

	return nil
}

func (r Request) Match(tx *bolt.Tx) (service, path, method *bolt.Bucket, err error) {
	service, err = r.ServiceBucket(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	pb := service.Bucket([]byte("paths"))
	if pb == nil {
		return service, nil, nil, nil
	}

	c := pb.Cursor()
	for key, _ := c.First(); key != nil; key, _ = c.Next() {
		if string(key) == r.Path {
			path = pb.Bucket(key)
			break
		}
	}

	if path == nil {
		return service, path, nil, nil
	}

	method = path.Bucket([]byte(r.Method))

	return service, path, method, nil
}
