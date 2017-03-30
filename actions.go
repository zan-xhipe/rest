package main

import (
	"strconv"
	"strings"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func setPath(ctx *kingpin.ParseContext) error {
	for _, c := range ctx.Elements {
		clause, ok := c.Clause.(*kingpin.ArgClause)

		if ok && clause.Model().Name == "path" {
			path = clause.Model().Value.String()
			return nil
		}
	}

	return nil
}

func setData(ctx *kingpin.ParseContext) error {
	for _, c := range ctx.Elements {
		clause, ok := c.Clause.(*kingpin.ArgClause)

		if ok && clause.Model().Name == "data" {
			data = strings.NewReader(clause.Model().Value.String())
			return nil
		}
	}

	return nil
}

func setService(ctx *kingpin.ParseContext) error {
	for _, c := range ctx.Elements {
		switch clause := c.Clause.(type) {
		case *kingpin.ArgClause:
			if clause.Model().Name == "service" {
				service = clause.Model().Value.String()
				return nil
			}

		case *kingpin.FlagClause:
			if clause.Model().Name == "service" {
				service = clause.Model().Value.String()
				return nil
			}
		}
	}

	return nil
}

func setNoHeaders(ctx *kingpin.ParseContext) error {
	for _, c := range ctx.Elements {
		clause, ok := c.Clause.(*kingpin.FlagClause)

		if ok && clause.Model().Name == "no-headers" {
			v, err := strconv.ParseBool(*c.Value)
			if err != nil {
				return err
			}

			noHeaders = v

			return nil
		}
	}

	return nil
}
