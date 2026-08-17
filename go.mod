module github.com/pj-hoakari/go-service-template

go 1.26.3

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	github.com/pj-hoakari/protoc-gen-authz-go/cmd/protoc-gen-authz-go
	go.uber.org/mock/mockgen
	google.golang.org/protobuf/cmd/protoc-gen-go
)

require (
	connectrpc.com/connect v1.20.0
	google.golang.org/protobuf v1.36.12
)

require github.com/pj-hoakari/protoc-gen-authz-go v0.2.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	go.uber.org/mock v0.6.0
)

require (
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/tools v0.36.0 // indirect
)
