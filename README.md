rest is a command line client for using web services.

to get it run
```go get -u github.com/zan-xhipe/rest```

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

To list all your services use ```rest service list```

You can switch between which service to use with ```rest service use <service-name>```

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

To view your setting for the current service use ```rest service config```

```
rest service set --header "Authorization=Bearer :token"
rest service set <path> --parameter token=tahonteoautnhanu
rest service set <path> <method> --scheme http --port 80
```

To remove a setting use ```rest service unset``` followed by the key to unset.  The key is a hierarchy that is '.' separated.  This lets you easily remove entire buckets, or individual settings e.g. ```rest service unset paths.users.get``` to unset a method specific setting, or ```rest service unset paths``` to remove all your path specific settings.

### scheme
	How to access the service, default is set to https.
### host
	Where the API is hosted, defaults to localhost.
### port
	Port to use when accessing the API, defaults to 443.
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

# Example
## Github
Github requires that you provide the accept header for the version of the API. ```$GITHUB_AUTH_TOKEN``` is a developer token.  We store the github username in ```$GITHUB_USER``` so that it can be set as a parameter.

Note the quotes when setting the authorization token, if we don't provide this it will split the header on the space and you will end up creating a path setting named after your token with a Authorization header that only contains ```token```.

Most of this setup could be done in the init command.  It is done as it is for illustrative proposes.

```
rest service init github \
	--host api.github.com \
	--header Accept=application/vnd.github.v3+json
rest service use github
rest service set --header Authorization="token $GITHUB_AUTH_TOKEN"
GITHUB_USER=$(rest get user --pretty --filter login)
rest service set --parameter user="$GITHUB_USER"
```

Retrieve information about yourself
```
rest get user
```

Retrieve all your repos
```
rest service alias repos get users/:user/repos
rest perform repos --pretty
```

# Todo
- Allow setting parameters from the filtered output of a request
- Make aliases first class commands instead of hiding them behind perform
- Figure out how to interact sensibly with hypermedia

# Motivation
This is mostly a tool to help me explore the APIs of various services that I intend to use, and interactively test the APIs I create for work.  Thus if there is anything missing it is because I have not had a need for it yet.  If you find this tool useful, great.  Issues and Pull Requests welcome.
