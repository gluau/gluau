ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

clean:
	rm -rf ./rustlib/rustlib/target
	rm ./rustlib/librustlib.a go-rust

library:
	$(MAKE) -C rustlib/rustlib build
	cp target/release/librustlib.a rustlib/librustlib.a

build:
	go build -o go-rust

all: library build

run: build
	./go-rust