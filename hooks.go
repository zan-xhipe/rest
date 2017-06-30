package main

import (
	"errors"
	"fmt"

	lua "github.com/yuin/gopher-lua"
	gopherjson "layeh.com/gopher-json"
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

	L, err := initLua()
	if err != nil {
		return err
	}

	r.ranHook = true

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

	L, err := initLua()
	if err != nil {
		return err
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

	L, err := initLua()
	if err != nil {
		return err
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

func initLua() (*lua.LState, error) {
	if luaState == nil {
		luaState = lua.NewState()
		gopherjson.Preload(luaState)
		if err := luaState.DoString(`json = require("json")`); err != nil {
			return nil, ErrHook{Context: "loading json helper", Err: err}
		}
		if err := luaState.DoString(luaHelpers); err != nil {
			return nil, ErrHook{Context: "loading table helpers", Err: err}
		}
	}
	return luaState, nil
}

var luaHelpers = `
-- found at https://svn.wildfiregames.com/public/ps/trunk/build/premake/premake4/src/base/table.lua
--
-- table.lua
-- Additions to Lua's built-in table functions.
-- Copyright (c) 2002-2008 Jason Perkins and the Premake project
--
	

--
-- Returns true if the table contains the specified value.
--

	function table.contains(t, value)
		for _,v in pairs(t) do
			if (v == value) then
				return true
			end
		end
		return false
	end
	
		
--
-- Enumerates an array of objects and returns a new table containing
-- only the value of one particular field.
--

	function table.extract(arr, fname)
		local result = { }
		for _,v in ipairs(arr) do
			table.insert(result, v[fname])
		end
		return result
	end
	
	

--
-- Flattens a hierarchy of tables into a single array containing all
-- of the values.
--

	function table.flatten(arr)
		local result = { }
		
		local function flatten(arr)
			for _, v in ipairs(arr) do
				if type(v) == "table" then
					flatten(v)
				else
					table.insert(result, v)
				end
			end
		end
		
		flatten(arr)
		return result
	end


--
-- Merges an array of items into a string.
--

	function table.implode(arr, before, after, between)
		local result = ""
		for _,v in ipairs(arr) do
			if (result ~= "" and between) then
				result = result .. between
			end
			result = result .. before .. v .. after
		end
		return result
	end


--
-- Returns true if the table is empty, and contains no indexed or keyed values.
--

	function table.isempty(t)
		return not next(t)
	end


--
-- Adds the values from one array to the end of another and
-- returns the result.
--

	function table.join(...)
		local result = { }
		for _,t in ipairs(arg) do
			if type(t) == "table" then
				for _,v in ipairs(t) do
					table.insert(result, v)
				end
			else
				table.insert(result, t)
			end
		end
		return result
	end


--
-- Return a list of all keys used in a table.
--

	function table.keys(tbl)
		local keys = {}
		for k, _ in pairs(tbl) do
			table.insert(keys, k)
		end
		return keys
	end


--
-- Translates the values contained in array, using the specified
-- translation table, and returns the results in a new array.
--

	function table.translate(arr, translation)
		local result = { }
		for _, value in ipairs(arr) do
			local tvalue
			if type(translation) == "function" then
				tvalue = translation(value)
			else
				tvalue = translation[value]
			end
			if (tvalue) then
				table.insert(result, tvalue)
			end
		end
		return result
	end

-- Print anything - including nested tables
function table.print (tt, indent, done)
  done = done or {}
  indent = indent or 0
  if type(tt) == "table" then
    for key, value in pairs (tt) do
      io.write(string.rep (" ", indent)) -- indent it
      if type (value) == "table" and not done [value] then
        done [value] = true
        io.write(string.format("[%s] => table\n", tostring (key)));
        io.write(string.rep (" ", indent+4)) -- indent it
        io.write("(\n");
        table.print (value, indent + 7, done)
        io.write(string.rep (" ", indent+4)) -- indent it
        io.write(")\n");
      else
        io.write(string.format("[%s] => %s\n",
            tostring (key), tostring(value)))
      end
    end
  else
    io.write(tt .. "\n")
  end
end

-- Returns number of  elements in the table
function table.length (tbl)
	local count = 0
	for _ in ipairs(tbl) do
		count = count + 1
	end
	return count
end
`
