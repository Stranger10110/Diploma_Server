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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func ReceiveFile(address string, filename string, nparts int) {
	if filename[:2] == "./" {
		filename = filename[2:]
	}

	doneStatuses := make(chan byte)
	var sum byte = 0
	go func() {
		for {
			sum = 0
			for sum < byte(nparts) {
				sum += <-doneStatuses
			}
			ConcatenateFiles(filename, nparts)
		}
	}()

	for i := 0; i < nparts; i++ {
		ln, err := net.Listen("tcp", address[:len(address)-1]+strconv.Itoa(i))
		if err != nil {
			log.Fatal(err)
		}

		go func(j int) {
			for {
				conn, err2 := ln.Accept()
				if err2 != nil {
					log.Println(err2)
					continue
				}
				// fmt.Println(conn.RemoteAddr())

				split := strings.Split(filename, ".")
				go receiveFile(conn, split[0]+"_part"+strconv.Itoa(j)+"."+split[1], doneStatuses)
			}
		}(i)
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

func receiveFile(tcp net.Conn, dstFile string, status chan byte) {
	defer tcp.Close()

	// read buffer size
	bufferSize := int(receiveInt64(tcp))
	// fmt.Printf("Buffer size: %d\n", bufferSize)
	bufferedTcp := bufio.NewReaderSize(tcp, bufferSize)

	// create new file
	fo, err := os.Create(dstFile)
	utils.CheckError(err, "receiveFile [1]", false)
	defer fo.Close()

	// read file size
	fileSize := receiveBufferedInt64(bufferedTcp)
	fmt.Printf("Chunk size: %d\n", fileSize)

	// accept file from client & write to new file
	_, err = io.CopyN(fo, bufferedTcp, fileSize)
	if err != nil && err.Error() != "EOF" {
		utils.CheckError(err, "receiveFile [2]", false)
	} else {
		if err != nil && err.Error() == "EOF" {
			fmt.Println("EOF")
		}
	}

	status <- byte(1)
}

func removeFilesByRegex(regex string) {
	files, err := filepath.Glob(regex)
	utils.CheckError(err, "removeFilesByRegex [1]", false)
	for _, f := range files {
		err = os.Remove(f)
		utils.CheckError(err, "removeFilesByRegex [2]", false)
	}
}

func ConcatenateFiles(filename string, nparts int) {
	split := strings.Split(filename, ".")
	files := split[0] + "_part*." + split[1]

	if nparts == 1 {
		part0 := split[0] + "_part0." + split[1]
		err := os.Rename(part0, filename)
		utils.CheckError(err, "ConcatenateFiles [2]", false)
		GetFileHash(filename)
		return
	}
	//var files string
	//for i := 0; i < nparts; i++ {
	//	files += " " + split[0]+"_part"+strconv.Itoa(i)+"."+split[1]
	//}

	cmd := exec.Command("bash", "-c", "cat "+files+" > "+filename)
	err := cmd.Run()
	utils.CheckError(err, "ConcatenateFiles [1]", false)

	removeFilesByRegex(files)
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
