.PHONY: build clean build-local

build: clean
	go build -o ./build/blaster

clean:
	rm -rf ./build

build-local: build
	cp ./build/blaster /usr/local/bin/