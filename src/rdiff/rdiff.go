package rdiff

//-m64 -mthreads -fmessage-length=0

/*
#cgo LDFLAGS: -L./ -lrsync_2_3_2 -Wl,-rpath,./
#include "./librsync.h"
#include <stdlib.h>
*/
import "C"
import (
	"../utils"

	"os"
	"unsafe"
)

var Rdiff *rdiff
var fileMode map[string]int

func init() {
	Rdiff = New()

	fileMode = make(map[string]int)
	fileMode["rb"] = os.O_RDONLY
	fileMode["rb+"] = os.O_RDWR
	fileMode["wb"] = os.O_CREATE | os.O_WRONLY
	fileMode["wb+"] = os.O_CREATE | os.O_RDWR
	fileMode["wob"] = os.O_WRONLY
	fileMode["wob+"] = os.O_RDWR
	fileMode["ab"] = os.O_APPEND | os.O_WRONLY

	// Rdiff.Signature("/home/stranger/Storages/Cloud/Seaweed/test2/Platform_designer_lab/заметки.txt", "/home/stranger/Storages/Cloud/Seaweed/Meta_test2/Platform_designer_lab/заметки.txt.sig.v1", "wb")
}

type rdiff struct{}

func New() *rdiff {
	// TODO: disable showing stats
	C.rdiff_set_params(0, 0, 1, 1)
	return &rdiff{}
}

func NewWithParams(block int, strong int, stats int, force int) *rdiff {
	C.rdiff_set_params(C.int(block), C.int(strong), C.int(stats), C.int(force))
	return &rdiff{}
}

func (r rdiff) Signature(filepath string, sigPath string, sigMode string) C.rs_result {
	basis, err := os.OpenFile(filepath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Signature() [1]", false)

	sig, err2 := os.OpenFile(sigPath, fileMode[sigMode], 0600)
	utils.CheckError(err2, "Rdiff.Signature() [2]", false)

	sigMode_ := C.CString(sigMode)
	defer C.free(unsafe.Pointer(sigMode_))

	res := C.rdiff_sig(C.int(basis.Fd()), C.int(sig.Fd()), sigMode_)
	if res != 0 {
		if err = basis.Close(); err != nil {
			return 12
		}
		if err = sig.Close(); err != nil {
			return 12
		}
	}
	return res
}

func (r rdiff) Signature2(filepath string, sigFd int, sigMode string) C.rs_result {
	basis, err := os.OpenFile(filepath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Signature2() [1]", false)

	sigMode_ := C.CString(sigMode)
	defer C.free(unsafe.Pointer(sigMode_))

	res := C.rdiff_sig(C.int(basis.Fd()), C.int(sigFd), sigMode_)
	if res != 0 {
		if err = basis.Close(); err != nil {
			return 12
		}
	}
	return res
}

func (r rdiff) Delta(sigPath string, newFilepath string, deltaPath string, deltaFileMode string) C.rs_result {
	sig, err := os.OpenFile(sigPath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Delta() [1]", false)

	new_, err2 := os.OpenFile(newFilepath, fileMode["rb"], 0600)
	utils.CheckError(err2, "Rdiff.Delta() [2]", false)

	delta, err3 := os.OpenFile(deltaPath, fileMode[deltaFileMode], 0600)
	utils.CheckError(err3, "Rdiff.Delta() [3]", false)

	deltaFileMode_ := C.CString(deltaFileMode)
	defer C.free(unsafe.Pointer(deltaFileMode_))

	res := C.rdiff_delta(C.int(sig.Fd()), C.int(new_.Fd()), C.int(delta.Fd()), deltaFileMode_)
	if res != 0 {
		if err = sig.Close(); err != nil {
			return 12
		}
		if err = new_.Close(); err != nil {
			return 12
		}
		if err = delta.Close(); err != nil {
			return 12
		}
	}
	return res
}

func (r rdiff) Delta2(sigFd int, newFilepath string, deltaPath string, deltaFileMode string) C.rs_result {
	new_, err := os.OpenFile(newFilepath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Delta2() [1]", false)

	delta, err2 := os.OpenFile(deltaPath, fileMode[deltaFileMode], 0600)
	utils.CheckError(err2, "Rdiff.Delta2() [2]", false)

	deltaFileMode_ := C.CString(deltaFileMode)
	defer C.free(unsafe.Pointer(deltaFileMode_))

	res := C.rdiff_delta(C.int(sigFd), C.int(new_.Fd()), C.int(delta.Fd()), deltaFileMode_)
	if res != 0 {
		if err = new_.Close(); err != nil {
			return 12
		}
		if err = delta.Close(); err != nil {
			return 12
		}
	}
	return res
}

func (r rdiff) Delta3(sigFd int, newFilepath string, deltaFd int, deltaFileMode string) C.rs_result {
	new_, err := os.OpenFile(newFilepath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Delta3() [1]", false)

	deltaFileMode_ := C.CString(deltaFileMode)
	defer C.free(unsafe.Pointer(deltaFileMode_))

	res := C.rdiff_delta(C.int(sigFd), C.int(new_.Fd()), C.int(deltaFd), deltaFileMode_)
	if res != 0 {
		if err = new_.Close(); err != nil {
			return 12
		}
	}
	return res
}

func (r rdiff) Patch(filepath string, deltaPath string, newFilepath string, newFileMode string) C.rs_result {
	basis, err := os.OpenFile(filepath, fileMode["rb"], 0600)
	utils.CheckError(err, "Rdiff.Patch() [1]", false)

	delta, err2 := os.OpenFile(deltaPath, fileMode["rb"], 0600)
	utils.CheckError(err2, "Rdiff.Patch() [2]", false)

	new_, err3 := os.OpenFile(newFilepath, fileMode[newFileMode], 0600)
	utils.CheckError(err3, "Rdiff.Patch() [3]", false)

	newMode := C.CString(newFileMode)
	defer C.free(unsafe.Pointer(newMode))

	res := C.rdiff_patch(C.int(basis.Fd()), C.int(delta.Fd()), C.int(new_.Fd()), newMode)
	if res != 0 {
		if err = basis.Close(); err != nil {
			return 12
		}
		if err = delta.Close(); err != nil {
			return 12
		}
		if err = new_.Close(); err != nil {
			return 12
		}
	}
	return res
}

/*func test() {
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
}*/
