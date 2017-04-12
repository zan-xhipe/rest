package main

import kingpin "gopkg.in/alecthomas/kingpin.v2"

var (
	_      = kingpin.Command("version", "display version info")
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

func requestFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("service", "the service to use").StringVar(&requestPath)
	cmd.Flag("no-headers", "ignore stored service headers").BoolVar(&noHeaders)

	settings.Flags(cmd)

	cmd.Flag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]'").StringVar(&filter)

}

func requestCommand(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&requestPath)
	requestFlags(cmd)
}

func requestDataCommand(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&requestPath)
	cmd.Arg("data", "data to send in the request").Required().StringVar(&data)

	requestFlags(cmd)
}
