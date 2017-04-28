rest is a command line client for using web services.

# Usage
To use a service you first have to initialise it.  This lets you provide the host name, port, scheme, and headers to use when accessing the service.
```
rest service init <service-name> --host example.com --header Content-Type=application/json
```

Once the service has been initialised you can perform common HTTP request on the service.  You should only provide the path you wish to request. GET, POST, PUT, and DELETE are supported.  If the returned data is json formatted then it can be filtered to display only specific parts of the returned data.
```
rest post users '{"username": "tester", "full_name": "Test Person"}'

rest get users
[{"id": 1, "username": "tester", "full_name": "Test Person"}]

rest get users --filter [0].username
"tester"
```

You can switch between which service to use with
```
rest service use <service-name>
```

# Aliases
You can set an alias for a specific call with ```rest service alias <name> <method> <path>``` you can then use ```rest perform <name> [<data>]``` to call the aliased path with the specified method.  This will use any path and method specific settings that you may have previously set.

This can be very useful to save actions that you perform often.  Combining this with parameters is especially useful in turning rest into a client for the service.

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
rest service set --parameter userID=1
rest get users/:userID
```

Parameters also work in headers and URL query items

# Headers
Providing the right headers is crucial to many requests.  This is also one of the main motivations for creating ```rest``` instead of using ```curl```.

# Filtering
You can filtered any returned json with a basic filter language, supported by gojq.
You can also pretty print output with the ```--pretty``` flag.  If you filter the output to a string you can remove the quotes around the string by also providing the pretty flag.

# Storing Settings
Various settings can be stored for each service you want to use.  Settings can be set per path, or per path access with a particular method.  By default settings apply to the current service.

Settings are stored in a boltdb database.  The default location is ```~/.restdb.db```

```
rest service set --header "Authorization=Bearer :token"
rest service set <path> --parameter token=tahonteoautnhanu
rest service set <path> <method> --scheme https --port 443
```

To remove a setting use ```rest service unset``` followed by the key to unset.  The key is a hierarchy that is '.' separated.  This lets you easily remove entire buckets, or individual settings e.g. ```rest service unset paths.users.get``` to unset a method specific setting, or ```rest service unset paths``` to remove all your path specific settings.

### scheme
	How to access the service, default is set to http.
### host
	Where the API is hosted, defaults to localhost.
### port
	Port to use when accessing the API, defaults to 80.
### base-path
	The base path is automatically appended to the host
### header
	Headers to send with the request, headers take the form of key=value
### query
	Query parameters that are sent with the request, set with key=value
### parameter
	Parameters can be substituted in other fields.

# Return Value
Because rest is intended to be used alongside other command line programs the HTTP response code returned by the service is mapped to a return value.  Any 200 response is mapped to 0, any 300 is mapped 3, 400 to 4, and 500 to 5. Errors resulting from bad input from the cli or errors in the service database return 1.
