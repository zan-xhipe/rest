package jsondata

import (
	"strconv"
	"strings"
)

func (jd *JSONData) Filter(exp string) (interface{}, error) {
	return jd.filter(jd.Data, exp)
}

func (jd *JSONData) filter(context interface{}, exp string) (interface{}, error) {
	if exp == "" {
		return context, nil
	}

	i := strings.Index(exp, ".")
	path := ""
	nexp := exp[i+1:]
	if i == -1 {
		i = len(exp)
		path = exp
		nexp = ""

	}
	path = exp[:i]

	switch {
	// results from the all elements in the array
	case path == "[]":
		v, ok := context.([]interface{})
		if !ok {
			return nil, ErrNotArray{Context: context}
		}

		arr := make([]interface{}, len(v))
		for index := range v {
			var err error
			arr[index], err = jd.filter(v[index], nexp)
			if err != nil {
				return nil, err
			}
		}

		return arr, nil
	// array index
	case strings.HasPrefix(path, "[") && strings.HasSuffix(path, "]"):
		index, err := strconv.Atoi(path[1 : len(path)-1])
		if err != nil {
			return nil, err
		}
		v, ok := context.([]interface{})
		if !ok {
			return nil, ErrNotArray{Context: context}
		}

		if len(v) <= index {
			return nil, ErrIndexOutOfRange{Index: index, Len: len(v)}
		}

		return jd.filter(v[index], nexp)
	// object
	default:
		v, ok := context.(map[string]interface{})
		if !ok {
			return nil, ErrNotObject{Context: context}
		}

		val, ok := v[path]
		if !ok {
			return nil, ErrNotExists{Path: path}
		}

		return jd.filter(val, nexp)
	}
}
