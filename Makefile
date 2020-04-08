.PHONY: build clean install test

TARGET=
SUFFIX=$(GOOS)_$(GOARCH)
ifneq ("${SUFFIX}", "_")
TARGET=_$(SUFFIX)
endif

build:
	CGO_ENABLED=0 go build -o "./build/blaster${TARGET}${EXTENSION}"

clean:
	rm -rf ./build

test: build
	CGO_ENABLED=0 go test -timeout 5s -covermode=count -coverpkg="blaster,blaster/core,blaster/cmd,blaster/sqs,blaster/kafka" -coverprofile=build/cover.out ./...

install: build
	cp "./build/blaster${TARGET}" /usr/local/bin/
