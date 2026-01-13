ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

library_normal:
	$(MAKE) -C rustlib/rustlib build_normal
	cp target/debug/librustlib.a rustlib/librustlib_linux_amd64.a

# Before running this, ensure you have the necessary targets installed:
# rustup default nightly # build_std needs nightly
# rustup target add x86_64-unknown-linux-gnu
# rustup target add aarch64-unknown-linux-gnu
# sudo apt install g++-aarch64-linux-gnu
# sudo apt-get install mingw-w64
distlib:
	$(MAKE) -C rustlib/rustlib build_linux_amd64
	cp target/x86_64-unknown-linux-gnu/release/librustlib.a rustlib/librustlib_linux_amd64.a
	$(MAKE) -C rustlib/rustlib build_linux_arm64
	cp target/aarch64-unknown-linux-gnu/release/librustlib.a rustlib/librustlib_linux_arm64.a
	$(MAKE) -C rustlib/rustlib build_windows_amd64
	cp target/x86_64-pc-windows-gnu/release/librustlib.a rustlib/librustlib_windows_amd64.a

distlib_musl:
	$(MAKE) -C rustlib/rustlib build_linux_amd64_musl
	cp target/x86_64-unknown-linux-musl/release/librustlib.a rustlib/librustlib_linux_amd64_musl.a

build:
	go build -o go-rust
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o go-rust-linux-amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o go-rust-linux-arm64
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o go-rust-windows-amd64.exe

all: library build

run: build
	./go-rust

clean:
	rm -rf ./rustlib/rustlib/target ./target
	rm ./rustlib/*.a go-rust