UNIT_TEST_PKG = $(shell go list ./... | grep -v /test)

.PHONY: unit-test
unit-test:
	go test -v -race $(UNIT_TEST_PKG)

.PHONY: compile-proto
compile-proto:
	protoc -I="./internal/work" --go_out="./internal" "./internal/work/work.proto"

.PHONY: local-test
local-test:
	go test -v -race ./test/local/...