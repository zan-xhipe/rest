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

	r.ranHook = true
	L := lua.NewState()
	defer L.Close()
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
