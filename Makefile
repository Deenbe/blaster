.PHONY: build clean

build: clean
		go build -o ./build/blaster

clean:
		rm -rf ./build