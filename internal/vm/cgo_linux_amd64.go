//go:build amd64 && linux

package vm

/*
#cgo LDFLAGS: -L../../rustlib -lrustlib -lstdc++ -lm -ldl
*/
import "C"
