package files

import (
	"../utils"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

type fileReader interface {
	Read(p []byte) (int, error)
}

//type wsReader struct {
//	ws *wsutil.Reader
//}

//func (w *wsReader) Read(p []byte) (int, error) {
//	_, err := w.ws.NextFrame()
//	utils.CheckError(err, "wsReader.Read [1]", false)
//	return w.ws.Read(p)
//}

func Exist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CreateDirIfNotExists(folderPath string) error {
	err := os.MkdirAll(folderPath, os.ModePerm)
	return err
}

//func CreateDirIfNotExists(dir string) {
//	if ok, err2 := Exist(dir); err2 == nil && !ok {
//		err3 := CreateDir(dir)
//		utils.CheckError(err3, "api.files.CreateDirIfNotExists() [1]", true)
//	} else if err2 != nil {
//		utils.CheckError(err2, "api.files.CreateDirIfNotExists() [2]", false)
//	}
//}

func CreateFileIfNotExists(filepath string) {
	if ok, err2 := Exist(filepath); err2 == nil && !ok {
		fo, err3 := os.Create(filepath)
		utils.CheckError(err3, "api.files.CreateFileIfNotExists() [1]", false)
		fo.Close()
	} else {
		utils.CheckError(err2, "api.files.CreateFileIfNotExists() [2]", false)
	}
}

func ReceiveFileTcp(address string, filename string) {
	if filename[:2] == "./" {
		filename = filename[2:]
	}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err2 := ln.Accept()
		if err2 != nil {
			log.Println(err2)
			continue
		}
		fmt.Println(conn.RemoteAddr())

		ReceiveFile(conn, filename)
		fmt.Printf("%s md5 checksum: %x\n", filename, GetFileMd5Hash(filename))
	}
}

func receiveInt64(reader fileReader) (result int64) {
	buff := make([]byte, 8)
	_, err := io.ReadFull(reader, buff)
	utils.CheckError(err, "receiveInt64 [1]", false)
	result = int64(binary.BigEndian.Uint64(buff))
	utils.CheckError(err, "receiveInt64 [2]", false)
	return
}

func ReceiveFile(conn net.Conn, dstFile string) {
	// read buffer size
	bufferSize := int(receiveInt64(conn))

	var fReader fileReader = bufio.NewReaderSize(conn, bufferSize)
	fReader = bufio.NewReaderSize(conn, bufferSize)

	// read file size
	fileSize := receiveInt64(fReader)
	fmt.Printf("Chunk size: %d\n", fileSize)

	// create new file
	fo, err := os.Create(dstFile)
	utils.CheckError(err, "receiveFile [1]", false)
	defer fo.Close()

	if fileSize == 0 {
		return
	}

	// accept file from client & write to new file
	_, err = io.CopyN(fo, fReader, fileSize)
	if err != nil && err.Error() != "EOF" {
		utils.CheckError(err, "receiveFile [2]", false)
	} else {
		if err != nil && err.Error() == "EOF" {
			fmt.Println("EOF")
		}
	}
}

func ReceiveFileIntoMemory(conn net.Conn) *bytes.Buffer {
	// read buffer size
	bufferSize := int(receiveInt64(conn))

	var fReader fileReader = bufio.NewReaderSize(conn, bufferSize)
	fReader = bufio.NewReaderSize(conn, bufferSize)

	// read file size
	fileSize := receiveInt64(fReader)
	fmt.Printf("Chunk size: %d\n", fileSize)

	// create new file
	memFile := bytes.NewBuffer(make([]byte, 0, fileSize))

	// accept file from client & write to new file
	_, err := io.CopyN(memFile, fReader, fileSize)
	if err != nil && err.Error() != "EOF" {
		utils.CheckError(err, "ReceiveFileIntoMemory [1]", false)
	} else {
		if err != nil && err.Error() == "EOF" {
			fmt.Println("EOF")
		}
	}

	return memFile
}
