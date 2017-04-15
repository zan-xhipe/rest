package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
)

var (
	defaultSettings = Settings{
		Scheme:       sql.NullString{String: "http", Valid: true},
		Host:         sql.NullString{String: "localhost", Valid: true},
		Port:         sql.NullInt64{Int64: 80, Valid: true},
		BasePath:     sql.NullString{String: "", Valid: true},
		Headers:      make(map[string]string),
		Parameters:   make(map[string]string),
		Queries:      make(map[string]string),
		Pretty:       sql.NullBool{Bool: false, Valid: true},
		PrettyIndent: sql.NullString{String: "\t", Valid: true},
	}
)

type Settings struct {
	Scheme   sql.NullString
	Host     sql.NullString
	Port     sql.NullInt64
	BasePath sql.NullString

	Headers    map[string]string
	Parameters map[string]string
	Queries    map[string]string

	Pretty       sql.NullBool
	PrettyIndent sql.NullString
}

func NewSettings() Settings {
	return Settings{
		Headers:    make(map[string]string),
		Parameters: make(map[string]string),
		Queries:    make(map[string]string),
	}
}

func (s *Settings) Merge(other Settings) {
	mergeString(&s.Scheme, other.Scheme)
	mergeString(&s.Host, other.Host)
	mergeInt(&s.Port, other.Port)
	mergeString(&s.BasePath, other.BasePath)
	mergeMap(s.Headers, other.Headers)
	mergeMap(s.Parameters, other.Parameters)
	mergeMap(s.Queries, other.Queries)
	mergeBool(&s.Pretty, other.Pretty)
	mergeString(&s.PrettyIndent, other.PrettyIndent)
}

func mergeString(a *sql.NullString, b sql.NullString) {
	if b.Valid {
		*a = b
	}
}

func mergeInt(a *sql.NullInt64, b sql.NullInt64) {
	if b.Valid {
		*a = b
	}
}

func mergeBool(a *sql.NullBool, b sql.NullBool) {
	if b.Valid {
		*a = b
	}
}

func mergeMap(a, b map[string]string) {
	for k, v := range b {
		a[k] = v
	}
}

func (s *Settings) Flags(cmd *kingpin.CmdClause) {
	cmd.Flag("scheme", "scheme used to access the service").
		Default(defaultSettings.Scheme.String).
		Action(usedFlag(&s.Scheme.Valid)).
		StringVar(&s.Scheme.String)

	cmd.Flag("host", "hostname for the service").
		Default(defaultSettings.Host.String).
		Action(usedFlag(&s.Host.Valid)).
		StringVar(&s.Host.String)

	cmd.Flag("port", "port to access the service").
		Default(fmt.Sprint(defaultSettings.Port.Int64)).
		Action(usedFlag(&s.Port.Valid)).
		Int64Var(&s.Port.Int64)

	cmd.Flag("base-path", "base path to use with service").
		Action(usedFlag(&s.BasePath.Valid)).
		StringVar(&s.BasePath.String)

	cmd.Flag("header", "set header for request").
		StringMapVar(&s.Headers)
	cmd.Flag("parameter", "set parameter for request").
		StringMapVar(&s.Parameters)
	cmd.Flag("query", "set query parameters for request").
		StringMapVar(&s.Queries)

	cmd.Flag("pretty", "pretty print json output").
		Action(usedFlag(&s.Pretty.Valid)).
		BoolVar(&s.Pretty.Bool)

	cmd.Flag("pretty-indent", "string to use to indent pretty json").
		Default(defaultSettings.PrettyIndent.String).
		Action(usedFlag(&s.PrettyIndent.Valid)).
		StringVar(&s.PrettyIndent.String)
}

func (s Settings) Write(b *bolt.Bucket) error {
	if b == nil {
		return errors.New("no bucket to write to")
	}
	if err := writeString(b, "scheme", s.Scheme); err != nil {
		return err
	}

	if err := writeString(b, "host", s.Host); err != nil {
		return err
	}

	if err := writeInt(b, "port", s.Port); err != nil {
		return err
	}

	if err := writeString(b, "base-path", s.BasePath); err != nil {
		return err
	}

	if err := writeMap(b, "headers", s.Headers); err != nil {
		return err
	}

	if err := writeMap(b, "parameters", s.Parameters); err != nil {
		return err
	}

	if err := writeMap(b, "queries", s.Queries); err != nil {
		return err
	}

	if err := writeBool(b, "pretty", s.Pretty); err != nil {
		return err
	}

	if err := writeString(b, "pretty-indent", s.PrettyIndent); err != nil {
		return err
	}

	return nil
}

func (s *Settings) Read(b *bolt.Bucket) {
	s.Scheme = readString(b, "scheme")
	s.Host = readString(b, "host")
	s.Port = readInt(b, "port")
	s.BasePath = readString(b, "base-path")
	bucketMap(b.Bucket([]byte("headers")), &s.Headers)
	bucketMap(b.Bucket([]byte("parameters")), &s.Parameters)
	bucketMap(b.Bucket([]byte("Queries")), &s.Queries)
	s.Pretty = readBool(b, "pretty")
	s.PrettyIndent = readString(b, "pretty-indent")
}

func (s Settings) Unset(b *bolt.Bucket) error {
	if s.Scheme.Valid {
		if err := b.Delete([]byte("scheme")); err != nil {
			return err
		}
	}

	if s.Host.Valid {
		if err := b.Delete([]byte("host")); err != nil {
			return err
		}
	}

	if s.Port.Valid {
		if err := b.Delete([]byte("port")); err != nil {
			return err
		}
	}

	if s.BasePath.Valid {
		if err := b.Delete([]byte("base-path")); err != nil {
			return err
		}
	}

	if err := unsetMapEntry(b, "headers", s.Headers); err != nil {
		return err
	}

	if err := unsetMapEntry(b, "parameters", s.Parameters); err != nil {
		return err
	}

	if err := unsetMapEntry(b, "queries", s.Queries); err != nil {
		return err
	}

	if s.Pretty.Valid {
		if err := b.Delete([]byte("pretty")); err != nil {
			return err
		}
	}

	if s.PrettyIndent.Valid {
		if err := b.Delete([]byte("pretty-indent")); err != nil {
			return err
		}
	}

	return nil
}

func (s Settings) URL() url.URL {
	u := url.URL{}
	u.Scheme = s.Scheme.String
	u.Host = fmt.Sprintf("%s:%d", s.Host.String, s.Port.Int64)
	return u
}

func LoadSettings(b *bolt.Bucket) Settings {
	s := NewSettings()
	s.Read(b)
	return s
}

func writeString(b *bolt.Bucket, key string, value sql.NullString) error {
	if !value.Valid {
		return nil
	}

	return b.Put([]byte(key), []byte(value.String))
}

func writeInt(b *bolt.Bucket, key string, value sql.NullInt64) error {
	if !value.Valid {
		return nil
	}

	buf := make([]byte, 4)
	binary.PutVarint(buf, value.Int64)
	return b.Put([]byte(key), buf)
}

func writeBool(b *bolt.Bucket, key string, value sql.NullBool) error {
	if !value.Valid {
		return nil
	}

	return b.Put([]byte(key), []byte(fmt.Sprint(value.Bool)))
}

func writeMap(b *bolt.Bucket, key string, data map[string]string) error {
	for k, v := range data {
		h, err := b.CreateBucketIfNotExists([]byte(key))
		if err != nil {
			return err
		}

		if err := h.Put([]byte(k), []byte(v)); err != nil {
			return err
		}
	}

	return nil
}

func readString(b *bolt.Bucket, key string) sql.NullString {
	v := b.Get([]byte(key))
	if v == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: string(v), Valid: true}
}

func readInt(b *bolt.Bucket, key string) sql.NullInt64 {
	v := b.Get([]byte(key))
	if v == nil {
		return sql.NullInt64{}
	}

	p, err := binary.ReadVarint(bytes.NewReader(v))
	if err != nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: p, Valid: true}
}

func readBool(b *bolt.Bucket, key string) sql.NullBool {
	v := b.Get([]byte(key))
	if v == nil {
		return sql.NullBool{}
	}

	p, err := strconv.ParseBool(string(v))
	if err != nil {
		return sql.NullBool{}
	}

	return sql.NullBool{Bool: p, Valid: true}
}

func unsetMapEntry(b *bolt.Bucket, key string, entries map[string]string) error {
	h := b.Bucket([]byte(key))
	if h == nil {
		return ErrMalformedDB{Bucket: key}
	}

	for key := range entries {
		if err := h.Delete([]byte(key)); err != nil {
			return err
		}
	}

	return nil
}
