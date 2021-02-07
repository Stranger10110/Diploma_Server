package threaded_handler

import (
	"../../utils"
	"bufio"
	"bytes"
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

func waitChannel(n int, sync chan int) {
	sum := 0
	for sum < n {
		sum += <-sync
	}
}

func ReceiveFile(address string, filename string, nparts int) {
	if filename[:2] == "./" {
		filename = filename[2:]
	}

	// create file
	fo, err := os.Create(filename)
	utils.CheckError(err, "ReceiveFileServer [1]", false)
	fo.Close()

	// appending to file
	file, err := os.OpenFile(filename, os.O_APPEND, 0644)
	utils.CheckError(err, "makeFile [1]", false)
	defer file.Close()

	sync1 := make(chan int)
	sync2 := make(chan int)
	fileParts := make([]*bytes.Buffer, nparts)

	// var sum byte = 0
	go func() {
		for {
			waitChannel(nparts, sync1)
			makeFile(file, &fileParts, nparts)

			for i := 0; i < nparts; i++ {
				sync2 <- 1
			}
		}
	}()

	for {
		ln, err := net.Listen("tcp", address)
		utils.CheckError(err, "ReceiveFile [2]", false)
		conn, err2 := ln.Accept()
		utils.CheckError(err2, "ReceiveFile [2]", false)
		// fmt.Println(conn.RemoteAddr())

		receiveFile(conn, &fileParts, sync1, sync2, nparts, address)
	}
}

func makeFile(file *os.File, parts *[]*bytes.Buffer, nparts int) {
	for i := 0; i < nparts; i++ {
		_, err := io.CopyN(file, (*parts)[i], int64((*parts)[i].Len()))
		if err != nil && err.Error() != "EOF" {
			utils.CheckError(err, "makeFile [1]", false)
		}
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

func receiveFile(tcp net.Conn, part *[]*bytes.Buffer, sync1 chan int, sync2 chan int, nparts int, address string) {
	defer tcp.Close()

	// read buffer size
	bufferSize := receiveInt64(tcp)
	// fmt.Printf("Buffer size: %d\n", bufferSize)
	fmt.Printf("Chunk size: %d\n", bufferSize)

	// read file size
	fileSize := receiveInt64(tcp)

	// read last bytes size
	lastBytes := int(receiveInt64(tcp))

	// create buffers
	for j := 0; j < nparts; j++ {
		b := make([]byte, bufferSize)
		(*part)[j] = bytes.NewBuffer(b)
		// (*part)[j] = &bytes.Buffer{}
	}

	// make threaded tcp connections
	threadedTcps := make([]net.Conn, nparts-1)
	bufferedTcps := make([]*bufio.Reader, nparts)
	bufferedTcps[0] = bufio.NewReaderSize(tcp, int(bufferSize))
	wait := make(chan int)
	var conn net.Conn
	var err2 error
	for j := 1; j < nparts; j++ {
		go func(j int) {
			ln, err := net.Listen("tcp", address[:len(address)-1]+strconv.Itoa(j))
			utils.CheckError(err, "receiveFile [1]", false)
			conn, err2 = ln.Accept()
			utils.CheckError(err2, "receiveFile [2]", false)
			fmt.Println(conn.LocalAddr())

			threadedTcps = append(threadedTcps, conn)
			bufferedTcps[j] = bufio.NewReaderSize(conn, int(bufferSize))
			wait <- 1
		}(j)
	}
	waitChannel(nparts-1, wait)

	// accept parts from client & write to buffers
	last := int64(lastBytes / nparts)
	last2 := fileSize - last*int64(nparts-1)
	for received := int64(0); received < fileSize; {
		for j := 0; j < nparts; j++ {
			go func(j int) {
				_, err := io.CopyN((*part)[j], bufferedTcps[j], bufferSize)
				if err != nil && err.Error() != "EOF" {
					utils.CheckError(err, "receiveFile [1]", false)
				} else {
					if err != nil && err.Error() == "EOF" {
						fmt.Println("EOF")
					}
				}

				sync1 <- 1
				received += bufferSize
				waitChannel(nparts, sync2)

				if (received >= (fileSize - bufferSize)) && (received <= fileSize) {
					bufferSize = last
					if j == nparts-1 {
						bufferSize = last2
					}
				}
			}(j)
		}
	}

	// close threaded connections
	for _, c := range threadedTcps {
		c.Close()
	}
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
