package rsync

//-m64 -mthreads -fmessage-length=0

/*
#cgo LDFLAGS: -L./ -lrsync -Wl,-rpath,./
#include "./librsync.h"
#include <stdlib.h>

import "C"*/

//basis := C.CString("test_data/koko.rar")
////new_ := C.CString("test_data/hello_new.docx")
//newCopy := C.CString("test_data/koko_copy.rar")
//sig := C.CString("test_data/koko.signature")
//delta := C.CString("test_data/koko.delta")
//defer C.free(unsafe.Pointer(basis))
//// defer C.free(unsafe.Pointer(new_))
//defer C.free(unsafe.Pointer(newCopy))
//defer C.free(unsafe.Pointer(sig))
//defer C.free(unsafe.Pointer(delta))

// C.rdiff_set_params(0, -1, 1, 1)

// C.rdiff_sig(basis, sig)
// C.rdiff_delta(sig, new_, delta)
// C.rdiff_patch(basis, delta, newCopy)
