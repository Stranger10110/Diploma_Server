package main

import (
	"./handlers"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	// "unsafe"
)

//-m64 -mthreads -fmessage-length=0

/*
#cgo LDFLAGS: -L./rsync -lrsync -Wl,-rpath,./rsync
#include "./rsync/librsync.h"
#include <stdlib.h>

import "C"*/

var (
	addr = flag.String("listen", "localhost:50000", "port to listen to")
)

func main() {
	fmt.Println("Server started...")
	_ = mux.NewRouter().StrictSlash(true) // router
	// sub := router.PathPrefix("/api").Subrouter()

	//sub.Methods("GET").Path("/ping_pong").HandlerFunc(handler.PingPong2)

	handlers.ReceiveFile(*addr, "./test_data/koko.rar")
	//sf := handler.NewSendFile(0.05)
	//sf.SendFile("./test_data/mek.delta", "192.168.0.244", 60000)
	//handler.GetFileHash("./test_data/mek.delta")

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

	// log.Fatal(http.ListenAndServe(":3000", router))
}
