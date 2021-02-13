package handler

import (
	"../utils"
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func ReceiveFile(address string, filename string) {
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
		// fmt.Println(conn.RemoteAddr())

		receiveFile(conn, filename)
		GetFileHash(filename)
	}
}

func receiveInt64(tcp net.Conn) (result int64) {
	buff := make([]byte, 8)
	_, err := io.ReadFull(tcp, buff)
	utils.CheckError(err, "receiveInt64 [1]", false)
	result = int64(binary.BigEndian.Uint64(buff))
	utils.CheckError(err, "receiveInt64 [2]", false)
	return
}

func receiveBufferedInt64(tcpBuffered *bufio.Reader) (result int64) {
	buff := make([]byte, 8)
	_, err := io.ReadFull(tcpBuffered, buff)
	utils.CheckError(err, "receiveBufferedInt64 [1]", false)
	result = int64(binary.BigEndian.Uint64(buff))
	utils.CheckError(err, "receiveBufferedInt64 [2]", false)
	return
}

func receiveFile(tcp net.Conn, dstFile string) {
	defer tcp.Close()

	// read buffer size
	bufferSize := int(receiveInt64(tcp))
	// fmt.Printf("Buffer size: %d\n", bufferSize)
	bufferedTcp := bufio.NewReaderSize(tcp, bufferSize)

	// read file size
	fileSize := receiveBufferedInt64(bufferedTcp)
	fmt.Printf("Chunk size: %d\n", fileSize)

	// create new file
	fo, err := os.Create(dstFile)
	utils.CheckError(err, "receiveFile [1]", false)
	defer fo.Close()

	// accept file from client & write to new file
	_, err = io.CopyN(fo, bufferedTcp, fileSize)
	if err != nil && err.Error() != "EOF" {
		utils.CheckError(err, "receiveFile [2]", false)
	} else {
		if err != nil && err.Error() == "EOF" {
			fmt.Println("EOF")
		}
	}
}

func GetFileHash(filename string) {
	// Open file for reading
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create new hasher, which is a writer interface
	hasher := md5.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		log.Fatal(err)
	}

	// Hash and print. Pass nil since
	// the data is not coming in as a slice argument
	// but is coming through the writer interface
	sum := hasher.Sum(nil)
	fmt.Printf("Md5 checksum: %x\n", sum)
}
