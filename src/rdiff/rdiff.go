package rdiff

//-m64 -mthreads -fmessage-length=0

/*
#cgo LDFLAGS: -L./ -lrsync -Wl,-rpath,./
#include "./librsync.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

var Rdiff *rdiff

func init() {
	Rdiff = New()
}

type rdiff struct{}

func New() *rdiff {
	C.rdiff_set_params(0, 0, 1, 1)
	return &rdiff{}
}

func NewWithParams(block int, strong int, stats int, force int) *rdiff {
	C.rdiff_set_params(C.int(block), C.int(strong), C.int(stats), C.int(force))
	return &rdiff{}
}

func (r rdiff) Signature(filepath string, sigPath string) C.rs_result {
	basis := C.CString(filepath)
	sig := C.CString(sigPath)

	defer C.free(unsafe.Pointer(basis))
	defer C.free(unsafe.Pointer(sig))

	return C.rdiff_sig(basis, sig)
}

func (r rdiff) Delta(sigPath string, newFilepath string, deltaPath string) C.rs_result {
	sig := C.CString(sigPath)
	new_ := C.CString(newFilepath)
	delta := C.CString(deltaPath)

	defer C.free(unsafe.Pointer(new_))
	defer C.free(unsafe.Pointer(sig))
	defer C.free(unsafe.Pointer(delta))

	return C.rdiff_delta(sig, new_, delta)
}

func (r rdiff) Patch(filepath string, deltaPath string, newFilepath string) C.rs_result {
	basis := C.CString(filepath)
	delta := C.CString(deltaPath)
	new_ := C.CString(newFilepath)

	defer C.free(unsafe.Pointer(basis))
	defer C.free(unsafe.Pointer(delta))
	defer C.free(unsafe.Pointer(new_))

	return C.rdiff_patch(basis, delta, new_)
}

func test() {
	home := "/home/stranger/"
	base := "Storages/Cloud/Seaweed/test2/Platform_designer_lab/"
	path := home + base

	basis := path + "Platform_designer_lab.docx"
	// new_ := path + "lab.docx"
	// newCopy := path + "lab_copy.docx"
	sig := home + "lab.sig"
	// delta := path + "lab.delta"

	Rdiff = New()
	fmt.Print("Sig result: ")
	fmt.Println(Rdiff.Signature(basis, sig))
}
