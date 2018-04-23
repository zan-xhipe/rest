package main

import (
	"os"
	"strings"
	"unicode"
)

func replacer(parameters map[string]string) func(string) string {
	params := paramReplacer(parameters)
	return func(input string) string {
		return os.ExpandEnv(params.Replace(input))
	}
}

func paramReplacer(parameters map[string]string) *strings.Replacer {
	rep := make([]string, 0, len(parameters))
	for key, value := range parameters {
		// colon parameter
		rep = append(rep, ":"+key)
		rep = append(rep, value)

		// bracket parameter
		rep = append(rep, "{"+key+"}")
		rep = append(rep, value)
	}

	return strings.NewReplacer(rep...)
}

func findParam(input string) string {
	out := ""

	if len(input) == 0 {
		return out
	}

	if input[0] == ':' {
		out = input[1:]

	}

	if input[0] == '{' && input[len(input)-1] == '}' {
		out = input[1 : len(input)-1]
	}

	return out

}

func paramFinder(input []string) []string {
	params := make([]string, 0)
	for _, p := range input {
		if param := findParam(p); param != "" {
			params = append(params, param)
		}
	}

	return params
}

func splitFunc(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	return unicode.IsSpace(r) ||
		r == ',' ||
		r == '"' ||
		r == '{' ||
		r == '}' ||
		r == '[' ||
		r == ']'
}

func dataSpliter(input string) []string {
	tmp := strings.FieldsFunc(input, splitFunc)
	result := make([]string, 0, len(tmp))
	for i := range tmp {
		if tmp[i] != ":" {
			result = append(result, tmp[i])
		}
	}
	return result
}
