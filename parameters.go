package main

import (
	"os"
	"regexp"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
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
		rep = append(rep, "{{"+key+"}}")
		rep = append(rep, value)
	}

	return strings.NewReplacer(rep...)
}

func addAliasParamsFromString(cmd *kingpin.CmdClause, name, str string) {
	for p, _ := range findParams(str) {
		addAliasParam(cmd, name, p)
	}
}

func findParams(input string) map[string]struct{} {
	re := regexp.MustCompile(`{{([[:word:]|-]*)}}|:[[:word:]]*`)
	matched := re.FindAllStringSubmatch(input, -1)
	params := make(map[string]struct{})
	for _, match := range matched {
		name := match[0]
		switch {
		case strings.HasPrefix(name, ":"):
			name = strings.TrimPrefix(name, ":")

		case strings.HasPrefix(name, "{{") && strings.HasSuffix(name, "}}"):
			name = strings.TrimPrefix(name, "{{")
			name = strings.TrimSuffix(name, "}}")

		}
		if name == "" {
			continue
		}
		params[name] = struct{}{}
	}

	return params
}
