package main

import (
	"errors"
	"fmt"

	lua "github.com/yuin/gopher-lua"
	"github.com/zan-xhipe/rest/internal/luahelpers"
)

var luaState *lua.LState

type ErrHook struct {
	Context string
	Err     error
}

func (e ErrHook) Error() string {
	return fmt.Sprintf("hook error during %s: %s", e.Context, e.Err)
}

func (r *Response) hook() error {
	if r.ResponseHook == "" {
		return nil
	}

	if luaState == nil {
		luaState = lua.NewState()
	}
	L := luaState

	r.ranHook = true

	if err := L.DoString(luahelpers.JSONLua); err != nil {
		return ErrHook{Context: "loading json helper", Err: err}
	}
	t := L.NewTable()
	t.RawSetString("status", lua.LNumber(r.resp.StatusCode))

	h := L.NewTable()
	for key, value := range r.resp.Header {
		v := L.NewTable()
		for i := range value {
			v.Append(lua.LString(value[i]))
		}
		h.RawSetString(key, v)
	}
	t.RawSetString("headers", h)

	t.RawSetString("body", lua.LString(string(r.Raw)))
	L.SetGlobal("response", t)

	if err := L.DoString(r.ResponseHook); err != nil {
		return ErrHook{Context: "perform response hook code", Err: err}
	}

	var ok bool
	t, ok = L.GetGlobal("response").(*lua.LTable)
	if !ok {
		return ErrHook{Context: "returning response", Err: errors.New("expected response to be a table")}
	}

	// you can only alter the response body
	// if for some reason there is a reasonable reason to need to alter the status code
	// or the headers, this should provide the starting point for getting those out of lua
	// status, ok := t.RawGetString("status").(lua.LNumber)
	// if !ok {
	// 	return ErrHook{Context: "returning response", Err: errors.New("expected number in status")}
	// }
	// r.resp.StatusCode, err = strconv.Atoi(status.String())
	// if err != nil {
	// 	return ErrHook{Context: "returning response", Err: err}
	// }

	// headers, ok := t.RawGetString("headers").(*lua.LTable)
	// if !ok {
	// 	return errors.New("expected a table in headers")
	// }
	// var err error
	// headers.ForEach(func(key, value lua.LValue) {
	// 	vt, ok := value.(*lua.LTable)
	// 	if !ok {
	// 		err = ErrHook{Context: "returning response", Err: errors.New("expected a table as header value")}
	// 		return
	// 	}
	// 	v := make([]string, 0, vt.Len())
	// 	vt.ForEach(func(_, h lua.LValue) {
	// 		v = append(v, h.String())
	// 	})

	// 	r.resp.Header[key.String()] = v
	// })
	// if err != nil {
	// 	return err
	// }

	r.display = []byte(t.RawGetString("body").String())
	return nil
}

func (r *Request) dataHook() error {
	if r.RequestDataHook == "" {
		return nil
	}

	if luaState == nil {
		luaState = lua.NewState()
	}
	L := luaState

	if err := L.DoString(luahelpers.JSONLua); err != nil {
		return ErrHook{Context: "loading json helper", Err: err}
	}
	L.SetGlobal("data", lua.LString(r.Data))
	if err := L.DoString(r.RequestDataHook); err != nil {
		return ErrHook{Context: "perform request data hook code", Err: err}
	}
	r.Data = L.GetGlobal("data").String()
	return nil
}

func (r *Request) hook() error {
	if r.RequestHook == "" {
		return nil
	}

	if luaState == nil {
		luaState = lua.NewState()
	}
	L := luaState

	if err := L.DoString(luahelpers.JSONLua); err != nil {
		return ErrHook{Context: "loading json helper", Err: err}
	}
	t := L.NewTable()
	t.RawSetString("path", lua.LString(r.URL.Path))
	t.RawSetString("data", lua.LString(r.Data))
	q := L.NewTable()
	for key, value := range r.URL.Query() {
		q.RawSetString(key, stringSliceToLua(L, value))
	}
	t.RawSetString("queries", q)
	h := L.NewTable()
	for key, value := range r.Settings.Headers {
		q.RawSetString(key, lua.LString(value))
	}
	t.RawSetString("headers", h)
	L.SetGlobal("request", t)

	if err := L.DoString(r.RequestHook); err != nil {
		return ErrHook{Context: "perform request hook code", Err: err}
	}

	var ok bool
	t, ok = L.GetGlobal("request").(*lua.LTable)
	if !ok {
		return ErrHook{Context: "returning request", Err: errors.New("expected request to be a table")}
	}
	r.URL.Path = t.RawGetString("path").String()
	r.Data = t.RawGetString("data").String()
	queries, ok := t.RawGetString("queries").(*lua.LTable)
	if !ok {
		return ErrHook{Context: "returning request", Err: errors.New("expeted a table in queries")}
	}

	r.URL.RawQuery = ""
	qi := r.URL.Query()
	queries.ForEach(func(key, value lua.LValue) {
		qi.Set(key.String(), value.String())
	})

	headers, ok := t.RawGetString("headers").(*lua.LTable)
	if !ok {
		return ErrHook{Context: "returning request", Err: errors.New("expected a table in headers")}
	}
	r.Settings.Headers = make(map[string]string)
	headers.ForEach(func(key, value lua.LValue) {
		r.Settings.Headers[key.String()] = value.String()
	})

	return nil
}

func stringSliceToLua(L *lua.LState, s []string) *lua.LTable {
	v := L.NewTable()
	for i := range s {
		v.Append(lua.LString(s[i]))
	}
	return v
}
