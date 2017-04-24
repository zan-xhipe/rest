rest is a command line client for using web services.

# Usage
To use a service you first have to initialise it.  This lets you provide the host name, port, scheme, and headers to use when accessing the service.
```
rest service init service-name --host example.com --header Content-Type=application/json
```

Once the service has been initialised you can perform common HTTP request on the service.  You should only provide the path you wish to request. GET, POST, PUT, and DELETE are supported.  If the returned data is json formatted then it can be filtered to display only specific parts of the returned data.
```
rest post users '{"username": "tester", "full_name": "Test Person"}'

rest get users
[{"id": 1, "username": "tester", "full_name": "Test Person"}]

rest get users --filter [0].username
"tester"
```

# Parameters
You can provide parameters with your request.  Parameters can either be stored in the service database using init, or provided with the request.  In a request the parameter name is preceded with ":", when storing the ":" is omitted.
```
rest get users/:userID --parameter userID=1
{"id": 1, "username": "tester", "full_name": "Test Person"}
```

Parameters can also be used in post data.
```
rest post users '{"username": ":username", "full_name": "Other Person"}' --parameter username=other
```

You can store parameters with other service settings.
```
rest service set service-name --parameter userID=1
rest get users/:userID
```

Parameters also work in headers and URL query items

# Return Value
Because rest is intended to be used alongside other command line programs the HTTP response code returned by the service is mapped to a return value.  Any 200 response is mapped to 0, any 300 is mapped 3, 400 to 4, and 500 to 5. Errors resulting from bad input from the cli or errors in the service database return 1.
