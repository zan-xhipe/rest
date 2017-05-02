package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/boltdb/bolt"
)

var (
	defaultSettings = Settings{
		Scheme:       sql.NullString{String: "https", Valid: true},
		Host:         sql.NullString{String: "localhost", Valid: true},
		Port:         sql.NullInt64{Int64: 443, Valid: true},
		BasePath:     sql.NullString{String: "", Valid: true},
		Headers:      make(map[string]string),
		Parameters:   make(map[string]string),
		Queries:      make(map[string]string),
		Username:     sql.NullString{String: "", Valid: true},
		Password:     sql.NullString{String: "", Valid: true},
		Pretty:       sql.NullBool{Bool: false, Valid: true},
		PrettyIndent: sql.NullString{String: "\t", Valid: true},
		Filter:       sql.NullString{String: "", Valid: true},
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

	// basic auth
	Username sql.NullString
	Password sql.NullString

	// output
	Pretty       sql.NullBool
	PrettyIndent sql.NullString
	Filter       sql.NullString
}

// NewSettings returns a initialised settings struct
func NewSettings() Settings {
	return Settings{
		Headers:    make(map[string]string),
		Parameters: make(map[string]string),
		Queries:    make(map[string]string),
	}
}

// Merge the provided settings into calling settings struct
func (s *Settings) Merge(other Settings) {
	mergeString(&s.Scheme, other.Scheme)
	mergeString(&s.Host, other.Host)
	mergeInt(&s.Port, other.Port)
	mergeString(&s.BasePath, other.BasePath)
	mergeMap(s.Headers, other.Headers)
	mergeMap(s.Parameters, other.Parameters)
	mergeMap(s.Queries, other.Queries)
	mergeString(&s.Username, other.Username)
	mergeString(&s.Password, other.Password)
	mergeBool(&s.Pretty, other.Pretty)
	mergeString(&s.PrettyIndent, other.PrettyIndent)
	mergeString(&s.Filter, other.Filter)
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

// Flags attach all the settings flags to a command
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

	cmd.Flag("username", "set basic auth username").
		Action(usedFlag(&s.Username.Valid)).
		StringVar(&s.Username.String)
	cmd.Flag("password", "set basic auth password, NOTE: stored in plain text").
		Action(usedFlag(&s.Password.Valid)).
		StringVar(&s.Password.String)

	cmd.Flag("pretty", "pretty print json output, removes quotes when filtering").
		Action(usedFlag(&s.Pretty.Valid)).
		BoolVar(&s.Pretty.Bool)

	cmd.Flag("pretty-indent", "string to use to indent pretty json").
		Default(defaultSettings.PrettyIndent.String).
		Action(usedFlag(&s.PrettyIndent.Valid)).
		StringVar(&s.PrettyIndent.String)

	cmd.Flag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]', for more filter options look at http://jmespath.org/ as filter uses JMESPath").
		Action(usedFlag(&s.Filter.Valid)).
		StringVar(&s.Filter.String)
}

// Write settings to the database
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

	if err := writeString(b, "username", s.Username); err != nil {
		return err
	}

	if err := writeString(b, "password", s.Password); err != nil {
		return err
	}

	if err := writeBool(b, "pretty", s.Pretty); err != nil {
		return err
	}

	if err := writeString(b, "pretty-indent", s.PrettyIndent); err != nil {
		return err
	}

	if err := writeString(b, "filter", s.Filter); err != nil {
		return err
	}

	return nil
}

// Read the settings from the database
func (s *Settings) Read(b *bolt.Bucket) {
	s.Scheme = readString(b, "scheme")
	s.Host = readString(b, "host")
	s.Port = readInt(b, "port")
	s.BasePath = readString(b, "base-path")
	bucketMap(b.Bucket([]byte("headers")), &s.Headers)
	bucketMap(b.Bucket([]byte("parameters")), &s.Parameters)
	bucketMap(b.Bucket([]byte("Queries")), &s.Queries)
	s.Username = readString(b, "username")
	s.Password = readString(b, "password")
	s.Pretty = readBool(b, "pretty")
	s.PrettyIndent = readString(b, "pretty-indent")
	s.Filter = readString(b, "filter")
}

// URL for the service
func (s Settings) URL() url.URL {
	u := url.URL{}
	u.Scheme = s.Scheme.String
	u.Host = fmt.Sprintf("%s:%d", s.Host.String, s.Port.Int64)
	return u
}

// LoadSettings from the database
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

	return b.Put([]byte(key), []byte(strconv.Itoa(int(value.Int64))))
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

	p, err := strconv.Atoi(string(v))
	if err != nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: int64(p), Valid: true}
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
