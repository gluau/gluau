//go:build cgo

package vm

/*
#cgo linux,amd64,!musl LDFLAGS: -L../rustlib -lrustlib_linux_amd64 -lstdc++ -lm -ldl -lpthread
#cgo linux,amd64,musl LDFLAGS: -L../rustlib -lrustlib_linux_amd64_musl -lstdc++ -lm -lpthread
#cgo linux,arm64 LDFLAGS: -L../rustlib -lrustlib_linux_arm64 -lstdc++ -lm -ldl -lpthread
#cgo windows,amd64 LDFLAGS: -L../rustlib -lrustlib_windows_amd64 -lstdc++ -lws2_32 -luserenv -lntdll
*/
import "C"
