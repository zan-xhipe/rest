package main

import kingpin "gopkg.in/alecthomas/kingpin.v2"

var (
	_ = kingpin.Command("version", "display version info").Hidden()

	srv     = kingpin.Command("service", "commands to manage service settings")
	initSrv = srv.Command("init", "initialise a service, flags can be used to stored a value for that particular setting.  These flags are available during all request where they set the value for that request.")
	remSrv  = srv.Command("remove", "remove a service")
	set     = srv.Command("set", "set a value, flags can be used to stored a value for that particular setting.  These flags are available during all request where they set the value for that request.")
	unset   = srv.Command("unset", "unset a value")
	use     = srv.Command("use", "switch service")
	lstSrv  = srv.Command("list", "list all stored services")
	config  = srv.Command("config", "show and alter service configs")
	action  = srv.Command("alias", "set an action")

	get    = kingpin.Command("get", "Perform a GET request")
	post   = kingpin.Command("post", "Perform a POST request")
	put    = kingpin.Command("put", "Perform a PUT request")
	patch  = kingpin.Command("patch", "Perform a PATCH request")
	delete = kingpin.Command("delete", "Perform a DELETE request")
	option = kingpin.Command("options", "Perform an OPTIONS request")
	head   = kingpin.Command("head", "Perform a HEAD request")
)

func init() {
	requestMethod(get)
	requestDataMethod(post)
	requestDataMethod(put)
	requestDataMethod(patch)
	requestMethod(delete)
	requestMethod(option)
	requestMethod(head)
}

// requestFlags apply to all the basic request types
func requestFlags(cmd *kingpin.CmdClause, hide bool) {
	s := cmd.Flag("service", "the service to use")
	if hide {
		s.Hidden()

	}
	s.StringVar(&request.Service)

	nh := cmd.Flag("no-headers", "ignore stored service headers")
	if hide {
		nh.Hidden()
	}
	nh.BoolVar(&request.NoHeaders)

	nq := cmd.Flag("no-queries", "ignore stored service queries")
	if hide {
		nq.Hidden()
	}
	nq.BoolVar(&request.NoQueries)

	settings = NewSettings()
	settings.Flags(cmd, hide)
}

// requestMethod applies to all requests that don't accept a body
func requestMethod(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&request.Path)
	requestFlags(cmd, false)
}

// requestDataMethod applies to all request that accept a body
func requestDataMethod(cmd *kingpin.CmdClause) {
	cmd.Arg("path", "url to perform request on").Required().StringVar(&request.Path)
	cmd.Arg("data", "data to send in the request").Required().StringVar(&request.Data)

	requestFlags(cmd, false)
}
