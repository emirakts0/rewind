package internal

import (
	"strings"
	"unsafe"
)

func Ptr[T any](v T) *T {
	return &v
}

func ptrToString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var b []byte
	p := unsafe.Pointer(ptr)
	for {
		val := *(*byte)(p)
		if val == 0 {
			break
		}
		b = append(b, val)
		p = unsafe.Pointer(uintptr(p) + 1)
	}
	return string(b)
}

func stringToPtr(s string) uintptr {
	if !strings.HasSuffix(s, "\x00") {
		s += "\x00"
	}
	b := []byte(s)
	return uintptr(unsafe.Pointer(&b[0]))
}
