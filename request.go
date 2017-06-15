package main

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

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

	Alias string

	URL url.URL

	verbose int
}

func (r *Request) Perform() (*http.Response, error) {
	if err := db.Update(request.LoadSettings); err != nil {
		return nil, err
	}

	req, err := r.Prepare()
	if err != nil {
		return nil, err
	}

	switch r.verbose {
	case 1:
		fmt.Println(request.URL.String())
	case 2, 3:
		// at level 3 display the raw request
		extra := false
		if r.verbose >= 3 {
			extra = true
		}

		dump, err := httputil.DumpRequestOut(req, extra)
		if err != nil {
			// this is only the verbose logging, so carry on in case of error
			fmt.Println(err)
			break
		}
		fmt.Println(string(dump))
	}

	return r.retry(req)
}

func (r *Request) retry(req *http.Request) (*http.Response, error) {
	client := &http.Client{}

	var resp *http.Response
	var err error
	maxAttempts := int(r.Settings.Retries.Int64) + 1

	for i := 0; i < maxAttempts; i++ {

		if r.verbose > 0 {
			fmt.Printf("attempt %d: %s %s\n", i, req.Method, req.URL)
		}

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// TODO: respect the Retry-After header
		delay := r.Settings.RetryDelay.Duration

		if r.Settings.ExponentialBackoff.Bool {
			delay *= time.Duration(math.Exp(float64(i)))
		}

		if r.Settings.RetryJitter.Bool {
			delay = time.Duration(rand.Intn(int(delay)))
		}

		if r.verbose > 0 && delay > 0 && i < maxAttempts {
			fmt.Printf("waiting %s to retry\n", delay)
		}
		<-time.After(delay)
	}

	return resp, err
}

// Prepare the http request.  This will substitute all the parameters,
// addd all the headers and query parameters
func (r *Request) Prepare() (*http.Request, error) {
	// prepare the url
	r.URL = r.Settings.URL()
	params := paramReplacer(r.Settings.Parameters)

	r.URL.Path = path.Join(r.Settings.BasePath.String, params.Replace(r.Path))
	r.Data = params.Replace(r.Data)

	if !r.NoQueries {
		q := r.URL.Query()
		for key, value := range r.Settings.Queries {
			v := params.Replace(value)
			if v[0] != ':' {
				q.Set(key, v)
			}
		}
		r.URL.RawQuery = q.Encode()
	}

	// don't send the body if it's empty
	var data io.Reader
	if r.Data != "" {
		data = strings.NewReader(r.Data)
	}

	// prepare the request
	req, err := http.NewRequest(
		strings.ToUpper(r.Method),
		r.URL.String(),
		data,
	)
	if err != nil {
		return nil, err
	}

	if r.Settings.Username.Valid && r.Settings.Password.Valid &&
		r.Settings.Username.String != "" && r.Settings.Password.String != "" {
		req.SetBasicAuth(r.Settings.Username.String, r.Settings.Password.String)
	}

	if !r.NoHeaders {
		for key, value := range r.Settings.Headers {
			v := params.Replace(value)
			if v[0] != ':' {
				req.Header.Set(key, v)
			}
		}
	}

	return req, nil
}

// MakeServiceBucket creates the bucket for the service
func (r *Request) MakeServiceBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
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

// ServiceBucket retrieves the bucket for the current requests service
func (r *Request) ServiceBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	if r.Service == "" {
		info := tx.Bucket([]byte("info"))
		if info == nil {
			return nil, ErrNoInfoBucket
		}
		current := info.Get([]byte("current"))
		r.Service = string(current)
	}

	if r.Service == "" {
		return nil, ErrNoServiceSet
	}

	sb := tx.Bucket([]byte("services"))
	if sb == nil {
		return nil, ErrNoServicesBucket
	}

	b := sb.Bucket([]byte(r.Service))
	if b == nil {
		return nil, ErrNoService{Name: r.Service}
	}

	return b, nil
}

// MakePathBucket creates the bucket for the path
func (r Request) MakePathBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
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

// PathBucket returns the bucket for the request path, creates if needed
func (r Request) PathBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	s, err := r.ServiceBucket(tx)
	if err != nil {
		return nil, err
	}

	pb := s.Bucket([]byte("paths"))
	if pb == nil {
		return nil, ErrNoPaths
	}

	b := pb.Bucket([]byte(r.Path))
	if b == nil {
		return nil, ErrInvalidPath{Path: r.Path}
	}

	return b, nil
}

// MakeMethodBucket returns the bucket for the request method, creates if needed
func (r Request) MakeMethodBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
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

// LoadSettings from the database
func (r *Request) LoadSettings(tx *bolt.Tx) error {
	sb, pb, mb, err := request.Match(tx)
	if err != nil {
		return err
	}

	// Start with blank settings
	r.Settings = NewSettings()

	// load service settings
	if sb != nil {
		r.Settings = LoadSettings(sb)
	}

	// load path settings
	if pb != nil {
		r.Settings.Merge(LoadSettings(pb))
	}

	// load method settings
	if mb != nil {
		r.Settings.Merge(LoadSettings(mb))
	}

	// load provided cli flags settings
	r.Settings.Merge(settings)

	return nil
}

// Match returns the relavant db buckets for all request settings, it will first check
// for a matching alias, then check generic paths, if there is a matching alias, it
// will be returned in the path bucket.
func (r Request) Match(tx *bolt.Tx) (service, path, method *bolt.Bucket, err error) {
	service, err = r.ServiceBucket(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	// match aliases first, if an alias matches then ignore path or method matches
	ab := service.Bucket([]byte("aliases"))
	if ab != nil {
		c := ab.Cursor()
		for key, _ := c.First(); key != nil; key, _ = c.Next() {
			if string(key) == r.Alias {
				path = ab.Bucket(key)
				return service, path, nil, nil
			}
		}
	}

	// match path
	pb := service.Bucket([]byte("paths"))
	if pb == nil {
		return service, nil, nil, nil
	}

	// Match the path, this will eventually be expanded to better match
	// the path.  So specific paths will be matched before generic ones
	// at the moment it requires an exact match
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
