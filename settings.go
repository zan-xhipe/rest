package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"

	"github.com/boltdb/bolt"
)

type NullDuration struct {
	Duration time.Duration
	Valid    bool
}

var (
	defaultSettings = Settings{
		Scheme:     sql.NullString{String: "https", Valid: true},
		Host:       sql.NullString{String: "localhost", Valid: true},
		Port:       sql.NullInt64{Int64: 443, Valid: true},
		BasePath:   sql.NullString{String: "", Valid: true},
		Headers:    make(map[string]string),
		Parameters: make(map[string]string),
		Queries:    make(map[string]string),
		Username:   sql.NullString{String: "", Valid: true},
		Password:   sql.NullString{String: "", Valid: true},

		Pretty:        sql.NullBool{Bool: false, Valid: true},
		PrettyIndent:  sql.NullString{String: "\t", Valid: true},
		Filter:        sql.NullString{String: "", Valid: true},
		SetParameters: make(map[string]string),

		ResponseHook:    sql.NullString{String: "", Valid: true},
		RequestDataHook: sql.NullString{String: "", Valid: true},
		RequestHook:     sql.NullString{String: "", Valid: true},

		Retries:            sql.NullInt64{Int64: 2, Valid: true},
		RetryDelay:         NullDuration{Duration: 100000000, Valid: true},
		ExponentialBackoff: sql.NullBool{Bool: true, Valid: true},
		RetryJitter:        sql.NullBool{Bool: true, Valid: true},
	}

	yamlFile *string
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
	Pretty        sql.NullBool
	PrettyIndent  sql.NullString
	Filter        sql.NullString
	SetParameters map[string]string

	// hooks
	ResponseHook    sql.NullString
	RequestDataHook sql.NullString
	RequestHook     sql.NullString

	Retries            sql.NullInt64
	RetryDelay         NullDuration
	ExponentialBackoff sql.NullBool
	RetryJitter        sql.NullBool
}

type YAMLSettings struct {
	Settings YAMLServiceSettings          `yaml:",inline"`
	Aliases  map[string]YAMLAliasSettings `yaml:"aliases"`
	Paths    map[string]struct {
		Settings YAMLServiceSettings            `yaml:",inline"`
		Methods  map[string]YAMLServiceSettings `yaml:",inline"`
	} `yaml:"paths"`
}

type YAMLAliasSettings struct {
	Settings    YAMLServiceSettings `yaml:",inline"`
	Description *string             `yaml:"description,omitempty"`
	Path        string              `yaml:"path"`
	Method      string              `yaml:"method"`
	Data        *string             `yaml:"data,omitempty"`
}

type YAMLServiceSettings struct {
	Scheme      *string           `yaml:"scheme,omitempty"`
	Host        *string           `yaml:"host,omitempty"`
	Port        *int              `yaml:"port,omitempty"`
	BasePath    *string           `yaml:"base-path,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	Queries     map[string]string `yaml:"queries,omitempty"`
	Username    *string           `yaml:"username,omitempty"`
	Password    *string           `yaml:"password,omitempty"`
	Parameters  map[string]string `yaml:"parameters,omitempty"`
	DataHook    *string           `yaml:"data-hook,omitempty"`
	RequestHook *string           `yaml:"request-hook,omitempty"`

	Output *YAMLOutputSettings `yaml:"output,omitempty"`

	Retry *YAMLRetrySettings `yaml:"retry,omitempty"`
}

type YAMLOutputSettings struct {
	Pretty              *bool             `yaml:"pretty,omitempty"`
	Indent              *string           `yaml:"indent,omitempty"`
	Filter              *string           `yaml:"filter,omitempty"`
	Hook                *string           `yaml:"hook,omitempty"`
	SetFilterParameters map[string]string `yaml:"set-filter-parameters,omitempty"`
	SetLuaParameters    map[string]string `yaml:"set-lua-parameters,omitempty"`
}

type YAMLRetrySettings struct {
	Retries            *int           `yaml:"retries,omitempty"`
	Delay              *time.Duration `yaml:"delay,omitempty"`
	ExponentialBackoff *bool          `yaml:"exponential-backoff,omitempty"`
	Jitter             *bool          `yaml:"jitter,omitempty"`
}

func WriteYAMLSettings(filename *string, db *DB, r *Request) error {
	yamlSettings, err := LoadYAMLSettings(*yamlFile)
	if err != nil {
		return err
	}

	return yamlSettings.Write(db, &request)
}

func LoadYAMLSettings(filename string) (*YAMLSettings, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var s YAMLSettings

	if err := yaml.Unmarshal(buf, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *YAMLSettings) Write(db *DB, r *Request) error {
	return db.Update(func(tx *bolt.Tx) error {
		info, services, err := db.Init(tx)
		if err != nil {
			return err
		}

		name := r.Service

		// reading yaml completely resets the service, so just delete and recreate it
		if err := services.DeleteBucket([]byte(name)); err != nil && err != bolt.ErrBucketNotFound {
			return err
		}
		sb, err := services.CreateBucket([]byte(name))
		if err != nil {
			return err
		}

		if err := s.Settings.Write(sb); err != nil {
			return err
		}

		// write aliases
		if s.Aliases != nil {
			a, err := sb.CreateBucket([]byte("aliases"))
			if err != nil {
				return err
			}

			for k, v := range s.Aliases {
				b, err := a.CreateBucket([]byte(k))
				if err != nil {
					return err
				}

				if err := setString(b, "description", v.Description); err != nil {
					return err
				}

				if err := setString(b, "path", &v.Path); err != nil {
					return err
				}

				if err := setString(b, "method", &v.Method); err != nil {
					return err
				}

				if err := setString(b, "data", v.Data); err != nil {
					return err
				}

				if err := v.Settings.Write(b); err != nil {
					return err
				}
			}

		}

		// write path/methods
		for k, v := range s.Paths {
			b, err := sb.CreateBucket([]byte(k))
			if err != nil {
				return err
			}

			if err := v.Settings.Write(b); err != nil {
				return err
			}

			for key, value := range v.Methods {
				m, err := b.CreateBucket([]byte(key))
				if err != nil {
					return err
				}

				if err := value.Write(m); err != nil {
					return err
				}
			}

		}

		// if there are no current service, set this one as current
		// this needs to happen last so that we don't set it if something goes wrong during creation
		if err := db.SetCurrentIfNotExists(info, name); err != nil {
			return err
		}

		return nil
	})
}

func (s *YAMLSettings) Read(b *bolt.Bucket) error {
	if err := s.Settings.Read(b); err != nil {
		return err
	}

	if aliasBucket := b.Bucket([]byte("aliases")); aliasBucket != nil {
		s.Aliases = make(map[string]YAMLAliasSettings)

		aliasBucket.ForEach(func(key, _ []byte) error {
			as := YAMLAliasSettings{}

			as.Settings.Read(aliasBucket)

			if buf := read(aliasBucket, "description"); buf != nil {
				desc := string(buf)
				as.Description = &desc
			}

			if buf := read(aliasBucket, "path"); buf != nil {
				as.Path = string(buf)
			}

			if buf := read(aliasBucket, "method"); buf != nil {
				as.Method = string(buf)
			}

			if buf := read(aliasBucket, "data"); buf != nil {
				data := string(buf)
				as.Data = &data
			}

			s.Aliases[string(key)] = as

			return nil
		})
	}

	return nil
}

func (s *YAMLServiceSettings) Write(b *bolt.Bucket) error {
	if err := write(b, "scheme", s.Scheme); err != nil {
		return err
	}

	if err := write(b, "host", s.Host); err != nil {
		return err
	}

	if err := write(b, "port", s.Port); err != nil {
		return err
	}

	if err := write(b, "base-path", s.BasePath); err != nil {
		return err
	}

	if err := writeMap(b, "headers", s.Headers); err != nil {
		return err
	}

	if err := writeMap(b, "queries", s.Queries); err != nil {
		return err
	}

	if err := write(b, "username", s.Username); err != nil {
		return err
	}

	if err := write(b, "password", s.Password); err != nil {
		return err
	}

	if err := writeMap(b, "parameters", s.Parameters); err != nil {
		return err
	}

	if err := write(b, "data-hook", s.DataHook); err != nil {
		return err
	}

	if err := write(b, "request-hook", s.RequestHook); err != nil {
		return err
	}

	if s.Output != nil {
		if err := write(b, "output.pretty", s.Output.Pretty); err != nil {
			return err
		}

		if err := write(b, "output.indent", s.Output.Indent); err != nil {
			return err
		}

		if err := write(b, "output.filter", s.Output.Filter); err != nil {
			return err
		}

		if err := writeMap(b, "output.set-filter-parameters", s.Output.SetFilterParameters); err != nil {
			return err
		}

		if err := writeMap(b, "output.set-lua-parameters", s.Output.SetLuaParameters); err != nil {
			return err
		}

		if err := write(b, "output.response-hook", s.Output.Hook); err != nil {
			return err
		}
	}

	if s.Retry != nil {
		if err := write(b, "retry.retries", s.Retry.Retries); err != nil {
			return err
		}

		if err := write(b, "retry.exponential-backoff", s.Retry.ExponentialBackoff); err != nil {
			return err
		}

		if err := write(b, "retry.delay", s.Retry.Delay); err != nil {
			return err
		}

		if err := write(b, "retry.jitter", s.Retry.Jitter); err != nil {
			return err
		}
	}
	return nil
}

func (s *YAMLServiceSettings) Read(b *bolt.Bucket) error {
	readString := func(key string, value *string) {
		if buf := read(b, key); buf != nil {
			b := string(buf)
			value = &b
		}
	}

	readInt := func(key string, value *int) {
		if buf := read(b, key); buf != nil {
			p, err := strconv.Atoi(string(buf))
			if err != nil {
				value = nil
			}

			value = &p
		}
	}

	readBool := func(key string, value *bool) {
		if buf := read(b, key); buf != nil {
			p, err := strconv.ParseBool(string(buf))
			if err != nil {
				value = nil
			}

			value = &p
		}
	}

	readDuration := func(key string, value *time.Duration) {
		if buf := read(b, key); buf != nil {
			d, err := time.ParseDuration(string(buf))
			if err != nil {
				value = nil
			}

			value = &d
		}
	}

	readMap := func(key string, value map[string]string) {
		bucketMap(getBucketFromBucket(b, key), &value)
	}

	readString("scheme", s.Scheme)
	readString("host", s.Host)
	readInt("port", s.Port)
	readString("base-path", s.BasePath)
	readMap("headers", s.Headers)
	readMap("queries", s.Queries)
	readString("username", s.Username)
	readString("password", s.Password)
	readMap("parameters", s.Parameters)
	readString("data-hook", s.DataHook)
	readString("request-hook", s.RequestHook)

	if b.Bucket([]byte("output")) != nil {
		s.Output = &YAMLOutputSettings{}
		readBool("output.pretty", s.Output.Pretty)
		readString("output.indent", s.Output.Indent)
		readString("output.filter", s.Output.Filter)
		readString("output.response-hook", s.Output.Hook)
		readMap("output.set-filter-parameters", s.Output.SetFilterParameters)
		readMap("output.set-lua-parameters", s.Output.SetLuaParameters)
	}

	if b.Bucket([]byte("retry")) != nil {
		s.Retry = &YAMLRetrySettings{}
		readInt("retry.retries", s.Retry.Retries)
		readBool("retry.exponential-backoff", s.Retry.ExponentialBackoff)
		readDuration("retry.delay", s.Retry.Delay)
		readBool("retry.jitter", s.Retry.Jitter)
	}

	return nil
}

// NewSettings returns a initialised settings struct
func NewSettings() Settings {
	return Settings{
		Headers:       make(map[string]string),
		Parameters:    make(map[string]string),
		Queries:       make(map[string]string),
		SetParameters: make(map[string]string),
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
	mergeMap(s.SetParameters, other.SetParameters)

	mergeString(&s.ResponseHook, other.ResponseHook)
	mergeString(&s.RequestDataHook, other.RequestDataHook)
	mergeString(&s.RequestHook, other.RequestHook)

	mergeInt(&s.Retries, other.Retries)
	mergeDuration(&s.RetryDelay, other.RetryDelay)
	mergeBool(&s.ExponentialBackoff, other.ExponentialBackoff)
	mergeBool(&s.RetryJitter, other.RetryJitter)
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

func mergeDuration(a *NullDuration, b NullDuration) {
	if b.Valid {
		*a = b
	}
}

// Flags attach all the settings flags to a command
func (s *Settings) Flags(cmd *kingpin.CmdClause, hide bool) {
	df := defaultSettings
	flg := func(name, usage, deflt string) *kingpin.FlagClause {
		c := cmd.Flag(name, usage)

		if deflt != "" {
			c = c.Default(deflt)
		}

		if hide {
			c = c.Hidden()
		}

		return c
	}

	stringFlag := func(name, usage, deflt string, v *sql.NullString) {
		flg(name, usage, deflt).Action(usedFlag(&v.Valid)).StringVar(&v.String)
	}

	intFlag := func(name, usage, deflt string, v *sql.NullInt64) {
		flg(name, usage, deflt).Action(usedFlag(&v.Valid)).Int64Var(&v.Int64)
	}

	mapFlag := func(name, usage string, m *map[string]string) {
		flg(name, usage, "").StringMapVar(m)
	}

	boolFlag := func(name, usage string, dflt bool, v *sql.NullBool) {
		flg(name, usage, "").Default(fmt.Sprint(dflt)).Action(usedFlag(&v.Valid)).BoolVar(&v.Bool)
	}

	durationFlag := func(name, usage string, dflt time.Duration, v *NullDuration) {
		flg(name, usage, "").Default(fmt.Sprint(dflt)).Action(usedFlag(&v.Valid)).DurationVar(&v.Duration)
	}

	stringFlag("scheme", "scheme used to access the service", df.Scheme.String, &s.Scheme)
	stringFlag("host", "hostname for the service", df.Host.String, &s.Host)
	intFlag("port", "port to access the service  on", strconv.Itoa(int(df.Port.Int64)), &s.Port)
	stringFlag("base-path", "base path to use with service", "", &s.BasePath)

	mapFlag("header", "set header for request", &s.Headers)
	mapFlag("parameter", "set parameter for request", &s.Parameters)
	mapFlag("query", "set query parameters for request", &s.Queries)

	stringFlag("username", "set basic auth username", "", &s.Username)
	stringFlag("password", "set basic auth password, NOTE: stored in plain text", "", &s.Password)

	boolFlag("pretty", "pretty print json output, removes quotes when filtering", df.Pretty.Bool, &s.Pretty)

	stringFlag("pretty-indent", "string to use to indent pretty json", df.PrettyIndent.String, &s.PrettyIndent)

	stringFlag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]', for more filter options look at http://jmespath.org/ as filter uses JMESPath", "", &s.Filter)

	mapFlag("set-parameter", "takes the form 'parameter.path=filter-expression' The parameter.path is a period separated path to the bucket where the parameter must be set.  filter-expression is a JMESPath expression that will be used to determine what the parameter is set to.  If the filter returns nothing, then the parameter is unset", &s.SetParameters)

	stringFlag("response-hook", "run lua script on response, happens before filtering", "", &s.ResponseHook)
	stringFlag("request-data-hook", "run lua script on request data, happens before parameter replacement", "", &s.RequestDataHook)
	stringFlag("request-hook", "run lua script on the entire request, happens after parameter replacement", "", &s.RequestHook)

	intFlag("retries", "how many times to retry the command if it fails", "", &s.Retries)
	durationFlag("retry-delay", "how long to wait between retries, accepts a duration", df.RetryDelay.Duration, &s.RetryDelay)
	boolFlag("exponential-backoff", "wether retries should exponentially backoff, uses the retry delay", df.ExponentialBackoff.Bool, &s.ExponentialBackoff)
	boolFlag("retry-jitter", "adds jitter to retry delay", df.RetryJitter.Bool, &s.RetryJitter)
}

func (s *Settings) YAMLFlag(cmd *kingpin.CmdClause) {
	yamlFile = cmd.Flag("yaml", "load settings from yaml file").String()
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

	if err := writeBool(b, "output.pretty", s.Pretty); err != nil {
		return err
	}

	if err := writeString(b, "output.indent", s.PrettyIndent); err != nil {
		return err
	}

	if err := writeString(b, "output.filter", s.Filter); err != nil {
		return err
	}

	if err := writeMap(b, "output.set-filter-parameters", s.SetParameters); err != nil {
		return err
	}

	if err := writeString(b, "output.response-hook", s.ResponseHook); err != nil {
		return err
	}

	if err := writeString(b, "request-hook", s.RequestDataHook); err != nil {
		return err
	}

	if err := writeString(b, "request-hook", s.RequestHook); err != nil {
		return err
	}

	if err := writeInt(b, "retry.retries", s.Retries); err != nil {
		return err
	}

	if err := writeDuration(b, "retry.delay", s.RetryDelay); err != nil {
		return err
	}

	if err := writeBool(b, "retry.exponential-backoff", s.ExponentialBackoff); err != nil {
		return err
	}

	if err := writeBool(b, "retry.jitter", s.RetryJitter); err != nil {
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
	bucketMap(b.Bucket([]byte("queries")), &s.Queries)
	s.Username = readString(b, "username")
	s.Password = readString(b, "password")
	s.Pretty = readBool(b, "output.pretty")
	s.PrettyIndent = readString(b, "output.indent")
	s.Filter = readString(b, "output.filter")
	bucketMap(b.Bucket([]byte("output.set-filter-parameters")), &s.SetParameters)
	s.ResponseHook = readString(b, "output.response-hook")
	s.RequestDataHook = readString(b, "data-hook")
	s.RequestHook = readString(b, "request-hook")

	s.Retries = readInt(b, "retry.retries")
	s.RetryDelay = readDuration(b, "retry.delay")
	s.ExponentialBackoff = readBool(b, "retry.exponential-backoff")
	s.RetryJitter = readBool(b, "retry.jitter")
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

func write(b *bolt.Bucket, key string, value interface{}) error {
	if value == nil || reflect.ValueOf(value).IsNil() {
		return nil
	}

	var err error
	v := fmt.Sprint(value)
	if r := reflect.ValueOf(value); r.Kind() == reflect.Ptr {
		v = fmt.Sprint(r.Elem())
	}

	k := strings.Split(key, ".")
	for i := range k {
		if i == len(k)-1 {
			return b.Put([]byte(k[len(k)-1]), []byte(v))
		}

		b, err = b.CreateBucketIfNotExists([]byte(k[i]))
		if err != nil {
			return err
		}
	}

	return fmt.Errorf("didn't put value %s into %s", v, key)
}

func writeString(b *bolt.Bucket, key string, value sql.NullString) error {
	if !value.Valid {
		return nil
	}

	return write(b, key, &value.String)
}

func writeInt(b *bolt.Bucket, key string, value sql.NullInt64) error {
	if !value.Valid {
		return nil
	}

	return write(b, key, &value.Int64)
}

func writeBool(b *bolt.Bucket, key string, value sql.NullBool) error {
	if !value.Valid {
		return nil
	}

	return write(b, key, &value.Bool)
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

func writeDuration(b *bolt.Bucket, key string, value NullDuration) error {
	if !value.Valid {
		return nil
	}

	v := fmt.Sprint(value.Duration)
	return write(b, key, &v)
}

func read(b *bolt.Bucket, key string) []byte {
	k := strings.Split(key, ".")
	last := len(k) - 1
	for i := range k {
		if i == last {
			return b.Get([]byte(k[last]))
		}

		b = b.Bucket([]byte(k[i]))
		if b == nil {
			return nil
		}
	}

	return nil
}

func readString(b *bolt.Bucket, key string) sql.NullString {
	v := read(b, key)
	if v == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: string(v), Valid: true}
}

func readInt(b *bolt.Bucket, key string) sql.NullInt64 {
	v := read(b, key)
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
	v := read(b, key)
	if v == nil {
		return sql.NullBool{}
	}

	p, err := strconv.ParseBool(string(v))
	if err != nil {
		return sql.NullBool{}
	}

	return sql.NullBool{Bool: p, Valid: true}
}

func readDuration(b *bolt.Bucket, key string) NullDuration {
	v := read(b, key)
	if v == nil {
		return NullDuration{}
	}

	d, err := time.ParseDuration(string(v))
	if err != nil {
		panic(err)
	}

	return NullDuration{Duration: d, Valid: true}
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
