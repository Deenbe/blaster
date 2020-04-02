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
	CGO_ENABLED=0 go test -covermode=count -coverpkg="blaster,blaster/lib,blaster/cmd" -coverprofile=build/cover.out ./...

install: build
	cp "./build/blaster${TARGET}" /usr/local/bin/
