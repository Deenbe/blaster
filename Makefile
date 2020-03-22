.PHONY: build clean build-local test

build: clean
	go build -o ./build/blaster

clean:
	rm -rf ./build

test: build
	go test -covermode=count -coverpkg="blaster,blaster/lib,blaster/cmd" -coverprofile=build/cover.out ./...

build-local: build
	cp ./build/blaster /usr/local/bin/
