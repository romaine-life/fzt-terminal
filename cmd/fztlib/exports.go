//go:build cgo

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/romaine-life/fzt/render"
)

// Every function taking text takes UTF-8 C strings; every function returning text returns a
// UTF-8 JSON string the caller frees with fzt_free.

//export fzt_open
func fzt_open(menu, options *C.char) *C.char {
	return C.CString(open(C.GoString(menu), C.GoString(options)))
}

//export fzt_key
func fzt_key(session C.int, event *C.char) *C.char {
	return C.CString(key(int(session), C.GoString(event)))
}

//export fzt_resize
func fzt_resize(session, width, height C.int) *C.char {
	return C.CString(resize(int(session), int(width), int(height)))
}

//export fzt_prompt
func fzt_prompt(session C.int, prompt_ *C.char) *C.char {
	return C.CString(prompt(int(session), C.GoString(prompt_)))
}

//export fzt_close
func fzt_close(session C.int) {
	closeSession(int(session))
}

//export fzt_version
func fzt_version() *C.char {
	return C.CString(render.Version)
}

//export fzt_free
func fzt_free(s *C.char) {
	C.free(unsafe.Pointer(s))
}

func main() {}
