.PHONY: build clean build-local test

TARGET=
SUFFIX=$(GOOS)_$(GOARCH)
ifneq ("${SUFFIX}", "_")
TARGET=_$(SUFFIX)
endif

build:
	go build -o "./build/blaster${TARGET}"

clean:
	rm -rf ./build

test: build
	go test -covermode=count -coverpkg="blaster,blaster/lib,blaster/cmd" -coverprofile=build/cover.out ./...

build-local: build
	cp "./build/blaster${TARGET}" /usr/local/bin/
