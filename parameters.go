package main

import "strings"

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
