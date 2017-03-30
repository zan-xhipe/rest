package main

func init() {
	set.Arg("service", "the service to use").Required().StringVar(&service)
	set.Flag("scheme", "scheme used to access the service").Default("http").Action(usedFlag(&usedScheme)).StringVar(&scheme)
	set.Flag("header", "header to set for each request").StringMapVar(&headers)
	set.Flag("host", "hostname for the service").Default("localhost").Action(usedFlag(&usedHost)).StringVar(&host)
	set.Flag("port", "port to access the service").Default("80").IntVar(&port)
}
