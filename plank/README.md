# Plank

## What is Plank?
Plank is a platform server that is the gateway for Bifröst-enabled services written in Golang that serve many
micro-services of Project EVE. It runs a Go HTTP server which can process simple HTTP requests directly, delegate them
to a Bifröst service and relay the results from the service back to the original requestor, or upgrade the connection
to WebSocket and let the client and service talk over the WebSocket.

## Build and run
### Build the project
At this stage we don't have any release so you'll need to build it yourself using Golang 1.13. Refer to the following
instructions and upon a successful build, look for the binary under `build/`.

```bash
# To build against your operating system
./scripts/build.sh

# To build against a specific operating system
GOOS=darwin|linux|windows ./scripts/build.sh
```

### Run the server
Currently there is only one command `start-server` so you would simply start the Plank server by typing
`./plank start-server`. See the following for all supported flags.

|Long flag|Short flag|Default value|Required|Description|
|----|----|----|----|----|
|--hostname|-n|localhost|false|Hostname where Plank is to be served. Also reads from `$PLANK_HOSTNAME` environment variable|
|--port|-p|30080|false|Port where Plank is to be served. Also reads from `$PLANK_PORT` environment variable|
|--rootDir|-r|<current directory>|false|Root directory for the server. Also reads from `$PLANK_ROOT` environment variable|
|--static|-s|-|false|Path where static files will be served|
|--enable-all-services|-a|false|false|When enabled all heavy services will be enabled (e.g. VMware Code API)|
|--no-fabric-broker|-|false|false|Do not start Fabric broker|
|--fabric-endpoint|-|/fabric|false|Fabric broker endpoint (ignored if --no-fabric-broker is present)|
|--topic-prefix|-|/topic|false|Topic prefix for Fabric broker (ignored if --no-fabric-broker is present)|
|--queue-prefix|-|/queue|false|Queue prefix for Fabric broker (ignored if --no-fabric-broker is present)|
|--request-prefix|-|/pub|false|Application request prefix for Fabric broker (ignored if --no-fabric-broker is present)|
|--request-queue-prefix|-|/pub/queue|false|Application request queue prefix for Fabric broker (ignored if --no-fabric-broker is present)|
|--shutdown-timeout|-|5|false|Graceful server shutdown timeout in minutes|
|--output-log|-l|stdout|false|Platform log output|
|--access-log|-l|stdout|false|HTTP server access log output|
|--error-log|-l|stderr|false|HTTP server error log output|
|--debug|-d|false|false|Debug mode|
|--no-banner|-b|false|false|Do not print Plank banner at startup|
|--prometheus|-|false|false|Enable Prometheus at /prometheus for metrics|

Examples are as follows:
```bash
# Start a server at 0.0.0.0:8080 without Fabric (WebSocket) broker
# Note that the default value for "localhost" will not let the server be
# reachable from external networks. In production, set it to your FQDN.
./plank start-server --no-fabric-broker --hostname 0.0.0.0 --port 8080

# Start a server with all types of logs printing to stdout/stderr
./plank start-server

# Start a server with a 10 minute graceful shutdown timeout
./plank start-server --shutdown-timeout 10

# Start a server with platform server logs printing to stdout and access/error logs to their respective files
./plank start-server --access-log server-access-$(date +%m%d%y).log --error-log server-error-$(date +%m%d%y).log

# Start a server with debug outputs enabled
./plank start-server -d

# Start a server without Plank splash banner
./plank start-server --no-banner

# Start a server with Prometheus enabled at /prometheus
./plank start-server --prometheus

# Start a server with static path served at `/static` for folder `static`
./plank start-server --static static 

# Start a server with static paths served at `/static` for folder `static` and
# at `/something` for folder `something`
./plank start-server --static static --static something

# Start a server with a predefined configuration file
./plank start-server --config-file config.json
```

## Advanced topics
### Authentication
Plank supports seamless integration with CSP for its OAuth 2.0 authentication flow. It
supports the authorization code grant for web applications and client credentials grant for server-to-server
applications. See below for the detailed guide for each flow.

#### Client Credentials flow
You'll choose this authentication flow when the Plank server needs to communicate with another
backend service essentially not requiring any interactions from the user during authentication. You will need to
create an OAuth 2.0 Client with `client_credentials` grant before following the steps below to implement the
authentication flow.

WIP

#### Authorization Code flow
You'll choose this authentication flow when the Plank server acts as an intermediary that processes
an authorization code returned from the CSP authorization server to obtain an access token. This is the process
you go through when logging into VMC Console or other VMware Cloud UIs. Similar with Client Credentials flow,
you will need to create an OAuth 2.0 Client with `authorization_code` grant before following the steps below to
implement the authentication flow.

WIP

### Writing a service and making it available through REST or WebSocket
Guide coming soon

## Questions, suggestions, 🕷 reports?
Slack: @kjosh, #eve-dev #project-eve-core (private channel)
