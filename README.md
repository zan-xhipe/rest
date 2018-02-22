rest is an adaptive command line client for using RESTish web services.

Sometimes what you want to do is just interact with a webservice from the CLI, usually this would be a job for curl, but including all the headers the service needs, and specifying all the query parameters, and having the full hostname and path on the in the same command make the commands long and difficult to read.

This tool lets you store all those settings so it can construct the request for you.  The data can also be parameterized to make it easy to change a part of the request.

The adaptive part of the description comes from being able to create aliases for requests that become subcommands of the tool, complete with flags to specify the parameters.  These aliases and their flags also become part of the tools usage text, accessible with the ```--help`` flag

It is also designed to interact nicely with other command line tools, as such is has inbuilt ability to filter any json response it receives using the [JMESPath](http://jmespath.org/) query language.

If you want to do more complex processing than json filtering you can run lua hooks at various points in the request/response process, these hooks can alter the data you send and display in arbitrary ways.

to get it run
```
go get -u github.com/zan-xhipe/rest
```

# Usage
To use a service you first have to initialise it.  This lets you provide the host name, port, scheme, and headers to use when accessing the service.
```
rest service init <service-name> --host example.com --header Content-Type=application/json
```

You can have multiple services and switch between them with ```rest service use <service-name>```

Once the service has been initialised you can perform common HTTP request on the service.  You should only provide the path you wish to request. GET, POST, PUT, DELETE, PATCH, OPTIONS, and HEAD are supported.  If the returned data is json formatted then it can be filtered to display only specific parts of the returned data.
```
rest post users '{"username": "tester", "full_name": "Test Person"}'

rest get users
[{"id": 1, "username": "tester", "full_name": "Test Person"}]

rest get users --filter [0].username
"tester"
```

To list all your services use ```rest service list```

# Aliases
You can set an alias for a specific call with ```rest service alias <name> <method> <path> [<description>]``` .  Aliases become top level subcommands directly under ```rest```.  Using help will display the aliases with their descriptions.  When calling an alias it will use and path and method specific settings that you may have previously set.  You can also store settings that only apply to that alias, in which case path and method settings will be ignored.

Any parameters in the aliased path will become flags in the aliased command that can be used to set that parameter when using the alias.

This can be very useful to save actions that you perform often.  Combining this with parameters and storing filters is especially useful in turning rest into a client for the service.

# Parameters
You can provide parameters with your request.  Parameters can either be stored in the service database using init, or provided with the request.  In a request the parameter name is preceded with ":", or surrounded by "{}". When storing the ":"/"{}" is omitted.
```
rest get users/{userID} --parameter userID=1
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

You can also store environment variables in parameters, these will be expanded when the request is made. If you don't quote the envar when you set it then it will expand when you set it instead of when you call it.  This can be very useful for setting secrets and not have them appear in the config. 
```
rest service set --parameter token='$SECRET_TOKEN'
rest service set --password='$PASSWORD'
rest service set --header Authorization='Token $TOKEN'
```

Parameters also work in headers and URL query items

# Headers
Providing the right headers is crucial to many requests.  This is also one of the main motivations for creating ```rest``` instead of using ```curl```.  You can provide headers when setting up the service with ```rest service init```, set them later with ```rest service set ```, or provide them with the request.  For all of these you pass the ```--header <key>=<value>``` flag.  To add multiple headers proved the flag multiple times.

# Filtering
You can filtered any returned json with the [JMESPath](http://jmespath.org/) json query language.  Use square brackets to address arrays and the the fields name to reference a field in an object.  There is a lot more that can be achieved with JMESPath such as match certain conditions, or trasforming the output, you should read the JMESPath documentation.

You can also pretty print output with the ```--pretty``` flag.  If you filter the output to a string you can remove the quotes around the string by also providing the pretty flag.

# Lua Hooks
You can process the returned response with lua scripts.  This allows you to perform more processing than just the JMESPath filtering will allow.  There are three places that your lua can be execute. ```response-hook``` is called once the response has been received but before any filtering has been applied.  The response is stored in the ```response``` table in lua.  It ```response.status``` stores the status code, ```response.headers``` contains all the headers, and ```response.body``` contains the response body.  store the output of your processing in ```response.body``` again for it to be displayed.  If you don't want to alter the response, but just want to output something along with the response you can print it from the lua hook.  This will appear before the response text.

The ```request-data-hook``` puts the provided post body into ```data``` string in lua.  If you want to affect the data sent put your result back in ```data```.  This hook runs before parameter replacement is done on the request body.

The ```request-hook``` allows gives you access to most parts of the request before it is made.  It puts the request in the ```request``` table.  ```request.path``` contains the path, ```request.data``` contains the post body, ```request.queries``` is a table containing the query parameters, and ```request.headers``` is a table containing the headers.  This hooks runs after parameter replacement.

All the lua hooks run in the same lua environment so if you can access previous hooks variables, however if the hook doesn't run then its data isn't populated.  If the hook is an empty string it will not run so to run a hook without doing anything pass it ```';'```.

## Lua helper functions
There are several lua helper functions preloaded into the lua hook environment.  If you think that other functions should be included, open an issue.  Ideally these will all eventually be written in gopher-lua

* ```json.decode(t)``` Turn json string into a lua table
* ```json.encode(t) ``` Turn lua table into json string
* ```table.contains(t, value)``` Returns true if the table contains the specified value.
* ```table.extract(arr, fname)```  Enumerates an array of objects and returns a new table containing only the value of one particular field.
* ```table.flatten(arr)``` Flattens a hierarchy of tables into a single array containing all of the values.
* ```table.implode(arr, before, after, between)``` Merges an array of items into a string.
* ```table.isempty(t)``` Returns true if the table is empty, and contains no indexed or keyed values.
* ```table.join(...)``` Adds the values from one array to the end of another and returns the result.
* ```table.keys(tbl)``` Return a list of all keys used in a table.
* ```table.translate(arr, translation)``` Translates the values contained in array, using the specified translation table, and returns the results in a new array.
* ```table.print(tt, indent, done)``` Print anything - including nested tables
* ```table.length(tbl)``` Returns number of  elements in the table

There are two helper functions already loaded in the lua environment ```json.decode``` and ```json.encode``` to make it easier to interact with json responses.  

# Storing Settings
Various settings can be stored for each service you want to use.  Settings can be set per path, per path access with a particular method, or per alias.  By default settings apply to the current service. 

Settings are stored in a boltdb database.  The default location is ```~/.restdb.db```

To view your setting for the current service use ```rest service config```

```
rest service set --header "Authorization=Bearer :token"
rest service set <path> --parameter token=<access-token>
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
### username
	HTTP basic auth username
### password
	HTTP basic auth password

## Set Parameter
You can use the ```--set-parameter``` flag to set a parameter from the output of the request.  It takes the path to the parameter bucket and a filter to apply to the response before it is stored.  The parameter path is a dotted string, if just the parameter is provided then it will be stored in the service top level settings, to store the parameter under a alias or a path/method you need to provide the path to that bucket.  For aliases this looks like ```aliases.<alias>``` for paths/methods ```paths.<path>[.<method>]```.  The filter is the same as the display filter.  If the filter returns no results then the parameter is unset.

This allows basic pagination with services that return an offset iterator.  Note though that in this case you will loop through the results if you don't check that the offset hasn't been set.

# Retries
Any request that could not be performed, or returns a 5XX status code will be retried twice with exponential backoff and jitter.  By default two retries are attempted, with 100ms exponential backoff and jitter.  To change this use.

```
rest service set --retries=10 --retry-delay=500ms --no-exponential-backoff --no-retry-jitter
```

# Return Value
Because rest is intended to be used alongside other command line programs the HTTP response code returned by the service is mapped to a return value.  Any 200 response is mapped to 0, any 300 is mapped 3, 400 to 4, and 500 to 5. Errors resulting from bad input from the cli or errors in the service database return 1.

# Example
	There are example configurations for some services in the examples/ directory.  To load these call ```rest service init <service> --yaml examples/<service>.yaml```  This will load all the settings from the file into the local database.  If you want to reload the example file just call it again.  Some of the examples will require you to set some parameters to work properly
## Github
Github requires that you provide the accept header for the version of the API. ```$GITHUB_AUTH_TOKEN``` is a developer token.  We store your Github usrename as the ```user``` parameter, it is called 'login' by Github.

You can load the github service using ```rest service init github --yaml examples/github.yaml```  You then need to call.

```
rest service use github
rest service set --parameter authtoken='$GITHUB_AUTH_TOKEN' \
	--parameter --user=<github-username> \
	--repo=<default-repo>
```

Below is an example of how to set up the github service without using the provided yaml file.

Note the quotes when setting the authorization token, if we don't provide this it will split the header on the space and you will end up creating a path setting named after your token with a Authorization header that only contains ```token```.

Most of this setup could be done in the init command.  It is done as it is for illustrative proposes.

```
rest service init github \
	--host api.github.com \
	--header Accept=application/vnd.github.v3+json
rest service use github
rest service set --header Authorization='token $GITHUB_AUTH_TOKEN'
rest get user --pretty --set-parameter user=login
```

Retrieve information about yourself
```
rest get user
```

Retrieve all your repos
```
rest get users/:user/repos
```

Create a alias to return a list of the usernames of everyone who has starred your repo.
```
rest service alias stargazers get repos/:user/:repo/stargazers \
	--filter [*].login \
	--description 'List all users who have starred :repo'

rest stargazers --repo <repo-name>
```

Now lets create an alias that will just tell us how many stars the repo has instead of listing everyone who starred it.  We will do this by using a lua hook.

```
rest service alias stars get repos/:user/:repo/stargazers \
	--description 'How many users have starred :repo' \
	--response-hook 'response.body = table.length(json.decode(response.body))'

rest stars --repo <repo-name>
```

# Todo
- Figure out how to interact sensibly with hypermedia

# Bash/ZSH Shell Completion
Rest is built on kingpin which provides bash/ZSH completion.  To enable completion add the following to your  bash_profile (or equivalent):

```eval "$(your-cli-tool --completion-script-bash)"```

Or for ZSH

```eval "$(your-cli-tool --completion-script-zsh)"```


# Motivation
This is mostly a tool to help me explore the APIs of various services that I intend to use, and interactively test the APIs I create for work. Thus if there is anything missing it is because I have not had a need for it yet.  If you find this tool useful, great.  Issues and Pull Requests welcome.
