package jsondata

import "fmt"

type ErrNotArray struct {
	Context interface{}
}

func (e ErrNotArray) Error() string {
	return fmt.Sprintf("%s is not an array", e.Context)
}

type ErrNotObject struct {
	Context interface{}
}

func (e ErrNotObject) Error() string {
	return fmt.Sprintf("%s is not an object", e.Context)
}

type ErrNotExists struct {
	Path string
}

func (e ErrNotExists) Error() string {
	return fmt.Sprintf("%s does not exist", e.Path)
}

type ErrIndexOutOfRange struct {
	Index int
	Len   int
}

func (e ErrIndexOutOfRange) Error() string {
	return fmt.Sprintf("%d is out of range %d", e.Index, e.Len)
}
