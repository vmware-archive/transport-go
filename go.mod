module github.com/vmware/transport-go

go 1.16

require (
	github.com/fatih/color v1.12.0
	github.com/go-stomp/stomp v2.1.4+incompatible
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli v1.22.5
	github.com/vmware/transport-go/plank v0.0.0-20210816220202-47a1f3e15fb8
)

replace github.com/vmware/transport-go/plank => ./plank
