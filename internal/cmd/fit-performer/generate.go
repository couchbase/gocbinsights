//go:generate sh -c "protoc --proto_path=./proto --go_out=./ --go_opt=module=github.com/couchbase/gocbinsights/internal/cmd/fit-performer --go-grpc_opt=module=github.com/couchbase/gocbinsights/internal/cmd/fit-performer --go-grpc_out=./  ./proto/*.proto"

package main
