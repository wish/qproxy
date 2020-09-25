PGG     := bin/protoc-gen-go
FFJ     := bin/ffjson
PGIT    := bin/protoc-go-inject-tag
PGGG    := bin/protoc-gen-grpc-gateway
PKGS    := $(shell go list ./... | grep -v vendor | grep -v rpc)

VERSION := $(shell git describe --tags 2> /dev/null || echo "unreleased")
V_DIRTY := $(shell git describe --exact-match HEAD 2> /dev/null > /dev/null || echo "-unreleased")
GIT     := $(shell git rev-parse --short HEAD)
DIRTY   := $(shell git diff-index --quiet HEAD 2> /dev/null > /dev/null || echo "-dirty")

GOFILES := $(shell find . -type f | grep go$$ ) rpc/qproxy.pb_ffjson.go rpc/qproxy.pb_jsonpb.go

default: build/qproxy.linux

build/qproxy.linux: ${GOFILES}
	@echo "$@"
	@GOOS=linux GOARCH=$(TARGETARCH) CGO_ENABLED=0 go build -o build/qproxy.linux -ldflags\
		"-X github.com/wish/qproxy.Version=$(VERSION)$(V_DIRTY) \
		 -X github.com/wish/qproxy.Git=$(GIT)$(DIRTY)" \
		github.com/wish/qproxy/cmd/qproxy

build/qproxy.darwin: ${GOFILES}
	@echo "$@"
	@GOOS=darwin CGO_ENABLED=0 go build -o build/qproxy.darwin -ldflags\
		"-X github.com/wish/qproxy.Version=$(VERSION)$(V_DIRTY) \
		 -X github.com/wish/qproxy.Git=$(GIT)$(DIRTY)" \
		github.com/wish/qproxy/cmd/qproxy

# all .go files are deps, so these are fine specified as such:
rpc/qproxy.pb.go: ${PGG} ${PGIT} rpc/qproxy.proto
	@echo "protoc $@"
	@protoc --plugin=${PGG} \
	        -I /usr/local/include -I.\
	        -I third_party/googleapis \
		-I rpc/ rpc/qproxy.proto \
		--go_out=plugins=grpc:.
	@sed s/,omitempty// $@ > $@.tmp
	@mv $@.tmp $@
	@${PGIT} -input=$@ 2> /dev/null

rpc/qproxy.pb.gw.go: ${PGGG} rpc/qproxy.proto
	@echo "protoc $@"
	@protoc -I /usr/local/include -I. \
		-I third_party/googleapis \
		--plugin=$(PGGG) \
		--grpc-gateway_out=logtostderr=true:. rpc/qproxy.proto

rpc/qproxy.pb_ffjson.go: ${FFJ} rpc/qproxy.pb.go rpc/qproxy.pb.gw.go
	@rm -f rpc/qproxy.pb_jsonpb.go
	bin/ffjson rpc/qproxy.pb.go

rpc/qproxy.pb_jsonpb.go: rpc/qproxy.pb_ffjson.go
	@cp rpc/qproxy.pb_jsonpb.go.stub rpc/qproxy.pb_jsonpb.go

.PHONY: coverage
coverage:
	@go test -coverprofile=/tmp/cover github.com/wish/qproxy
	@go tool cover -html=/tmp/cover -o coverage.html
	@rm /tmp/cover

# the reason we introduce test and vtest is to have a way to view the results
# of failing tests, but being silient in the successful case (silence is golden
# principle)
.PHONY: test
test: default
	@echo "running unittests"
	@go test -cover ${PKGS} > /dev/null

.PHONY: vtest
vtest: default
	@go test -cover -v ${PKGS}

.PHONY: clean
clean:
	rm -rf build
	rm -f rpc/qproxy.pb.go rpc/qproxy.pb.gw.go rpc/qproxy.pb_ffjson.go rpc/qproxy.pb_jsonpb.go
	rm -f bin/ffjson bin/protoc-gen-go bin/protoc-gen-grpc-gateway bin/protoc-go-inject-tag

.PHONY: release
release: default test build/checksums.256

build/checksums.256: build/qproxy.linux build/qproxy.darwin
	@rm -f build/checksums.256
	@cd build && shasum -a 256 * > checksums.256

$(PGG):
	@echo "$@"
	@go build -o $(PGG) ./vendor/github.com/golang/protobuf/protoc-gen-go

$(FFJ):
	@echo "$@"
	@go build -o $(FFJ) ./vendor/github.com/pquerna/ffjson

$(PGIT):
	@echo "$@"
	@go build -o $(PGIT) ./vendor/github.com/favadi/protoc-go-inject-tag

$(PGGG):
	@echo "$@"
	@go build -o $(PGGG) ./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

