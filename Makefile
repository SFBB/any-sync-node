.PHONY: proto build test deps
export GOPRIVATE=github.com/anytypeio
export PATH:=deps:$(PATH)

build:
	@$(eval FLAGS := $$(shell PATH=$(PATH) govvv -flags -pkg github.com/anytypeio/any-sync/app))
	go build -v -o bin/any-sync-node -ldflags "$(FLAGS)" github.com/anytypeio/any-sync-node/cmd

test:
	go test ./... --cover

proto:
	protoc --gogofaster_out=:. --go-drpc_out=protolib=github.com/gogo/protobuf:. nodesync/nodesyncproto/protos/*.proto
	protoc --gogofaster_out=:. --go-drpc_out=protolib=github.com/gogo/protobuf:. debug/nodedebugrpc/nodedebugrpcproto/protos/*.proto

deps:
	go mod download
	go build -o deps storj.io/drpc/cmd/protoc-gen-go-drpc
	go build -o deps/protoc-gen-gogofaster github.com/gogo/protobuf/protoc-gen-gogofaster
	go build -o deps github.com/ahmetb/govvv

