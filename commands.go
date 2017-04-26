package main

import kingpin "gopkg.in/alecthomas/kingpin.v2"

var (
	_ = kingpin.Command("version", "display version info")

	srv     = kingpin.Command("service", "manage service settings")
	initSrv = srv.Command("init", "initialise a service")
	set     = srv.Command("set", "set a value")
	unset   = srv.Command("unset", "unset a value")
	use     = srv.Command("use", "switch service")
	lstSrv  = srv.Command("list", "list all stored services")
	config  = srv.Command("config", "show and alter service configs")

	get    = kingpin.Command("get", "Perform a GET request")
	post   = kingpin.Command("post", "Perform a POST request")
	put    = kingpin.Command("put", "Perform a PUT request")
	delete = kingpin.Command("delete", "Perform a DELETE request")
)

func init() {
	requestMethod(get)
	requestDataMethod(post)
	requestDataMethod(put)
	requestMethod(delete)
}

// requestFlags apply to all the basic request types
func requestFlags(cmd *kingpin.CmdClause) {
	cmd.Flag("service", "the service to use").StringVar(&request.Service)
	cmd.Flag("no-headers", "ignore stored service headers").BoolVar(&request.NoHeaders)
	cmd.Flag("no-queries", "ignore stored service queries").BoolVar(&request.NoQueries)

	settings = NewSettings()
	settings.Flags(cmd)

	cmd.Flag("filter", "pull parts out of the returned json. use [#] to access specific elements from an array, use the key name to access the key. eg. '[0].id', 'id', and 'things.[1]'").StringVar(&filter)

}

// requestMethod applies to all requests that don't accept a body
func requestMethod(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&request.Path)
	requestFlags(cmd)
}

// requestDataMethod applies to all request that accept a body
func requestDataMethod(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&request.Path)
	cmd.Arg("data", "data to send in the request").Required().StringVar(&request.Data)

	requestFlags(cmd)
}
