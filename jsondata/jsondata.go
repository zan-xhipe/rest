package jsondata

import "encoding/json"

type JSONData struct {
	Data interface{}
}

func New(jsondata string) (*JSONData, error) {
	var data = new(interface{})
	if err := json.Unmarshal([]byte(jsondata), data); err != nil {
		return nil, err
	}

	return &JSONData{*data}, nil
}
