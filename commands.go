package main

import kingpin "gopkg.in/alecthomas/kingpin.v2"

var (
	set    = kingpin.Command("init", "initialise rest session")
	config = kingpin.Command("config", "show and alter service configs")
	use    = kingpin.Command("use", "switch service")

	get    = kingpin.Command("get", "Perform a GET request")
	post   = kingpin.Command("post", "Perform a POST request")
	put    = kingpin.Command("put", "Perform a PUT request")
	delete = kingpin.Command("delete", "Perform a DELETE request")
)

func init() {
	requestCommand(get)
	requestDataCommand(post)
	requestDataCommand(put)
	requestCommand(delete)
}

func requestCommand(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&path)
	cmd.Flag("service", "the service to use").StringVar(&path)
	cmd.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	cmd.Flag("header", "set header for request").StringMapVar(&headers)
	cmd.Flag("parameter", "set parameter for request").StringMapVar(&parameters)
	cmd.Flag("scheme", "scheme used to access the service").
		Default("http").
		Action(usedFlag(&usedScheme)).
		StringVar(&scheme)

	cmd.Flag("host", "hostname for the service").
		Default("localhost").
		Action(usedFlag(&usedHost)).
		StringVar(&host)

	cmd.Flag("port", "port to access the service").
		Default("80").
		Action(usedFlag(&usedPort)).
		IntVar(&port)

	cmd.Flag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]'").StringVar(&filter)

	cmd.Flag("pretty", "pretty print json output").BoolVar(&pretty)
	cmd.Flag("pretty-indent", "string to use to indent pretty json").
		Default("\t").
		Action(usedFlag(&usedPrettyIndent)).
		StringVar(&prettyIndent)

}

func requestDataCommand(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&path)
	cmd.Arg("data", "data to send in the request").Required().StringVar(&data)
	cmd.Flag("service", "the service to use").StringVar(&service)
	cmd.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)
	cmd.Flag("header", "set header for request").StringMapVar(&headers)
	cmd.Flag("parameter", "set parameter for request").StringMapVar(&parameters)
	cmd.Flag("scheme", "scheme used to access the service").
		Default("http").
		Action(usedFlag(&usedScheme)).
		StringVar(&scheme)

	cmd.Flag("host", "hostname for the service").
		Default("localhost").
		Action(usedFlag(&usedHost)).
		StringVar(&host)

	cmd.Flag("port", "port to access the service").
		Default("80").
		Action(usedFlag(&usedPort)).
		IntVar(&port)

	cmd.Flag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]'").StringVar(&filter)

	cmd.Flag("pretty", "pretty print json output").BoolVar(&pretty)
	cmd.Flag("pretty-indent", "string to use to indent pretty json").
		Default("\t").
		Action(usedFlag(&usedPrettyIndent)).
		StringVar(&prettyIndent)

}
