package common

import "unsafe"

func Str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func Bytes2str(b []byte) (s string) {
	if len(b) < 1 {
		return
	}
	s = *(*string)(unsafe.Pointer(&b))
	return
}
