.PHONY: build clean build-local test

build: clean
	go build -o ./build/blaster

clean:
	rm -rf ./build

test:
	go test ./...

build-local: build
	cp ./build/blaster /usr/local/bin/
