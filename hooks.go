package main

import (
	lua "github.com/yuin/gopher-lua"
	"github.com/zan-xhipe/rest/internal/jsonlua"
)

// hook allows you to execute lua code on the response, comes with a json library
// the response is stored in 'response' as a string, 'response' is then passed back
func hook(code, response string) (string, error) {
	if code == "" {
		return response, nil
	}

	L := lua.NewState()
	defer L.Close()

	if err := L.DoString(jsonlua.JSONLua); err != nil {
		return response, err
	}
	L.SetGlobal("response", lua.LString(response))
	if err := L.DoString(code); err != nil {
		return response, err
	}
	resp := L.GetGlobal("response")

	return resp.String(), nil
}
