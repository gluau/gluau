ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

clean:
	rm -rf ./rustlib/rustlib/target
	rm ./rustlib/librustlib.so go-rust

library:
	$(MAKE) -C rustlib/rustlib build

build:
	cp target/release/librustlib.so ./rustlib
	go build -ldflags="-r $(ROOT_DIR)rustlib" -o go-rust

all: library build

run: build
	./go-rust