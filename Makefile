.PHONY: all build test clean

build:
	build/build.sh build

clean:
	rm -rf _output

install:
	build/build.sh install

test:
	build/build.sh test

