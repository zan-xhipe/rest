package main

import (
	"strings"
)

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
