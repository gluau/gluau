build_linux_amd64:
	cargo build-std-release -vvv --target x86_64-unknown-linux-gnu
	x86_64-linux-gnu-strip --strip-debug ../../target/x86_64-unknown-linux-gnu/release/librustlib.a
build_linux_arm64:
	cargo build-std-release -vvv --target aarch64-unknown-linux-gnu
	aarch64-linux-gnu-strip --strip-debug ../../target/aarch64-unknown-linux-gnu/release/librustlib.a
build_windows_amd64:
	cargo build-std-release -vvv --target x86_64-pc-windows-gnu
	x86_64-w64-mingw32-strip --strip-debug ../../target/x86_64-pc-windows-gnu/release/librustlib.a

build_normal:
	cargo build -vvv

build_linux_amd64_musl:
	cargo build --release -vvv --target x86_64-unknown-linux-musl
	strip --strip-debug ../../target/x86_64-unknown-linux-musl/release/librustlib.a