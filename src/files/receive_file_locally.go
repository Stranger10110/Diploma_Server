package files

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
	"strconv"

	// "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type fileReader interface {
	Read(p []byte) (int, error)
}

type wsReader struct {
	ws *wsutil.Reader
}

func (w *wsReader) Read(p []byte) (int, error) {
	_, err := w.ws.NextFrame()
	utils.CheckError(err, "wsReader.Read [1]", false)
	return w.ws.Read(p)
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CreateDir(folderPath string) error {
	err := os.MkdirAll(folderPath, os.ModePerm)
	return err
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

		ReceiveFile(conn, filename, "buff")
		GetFileHash(filename)
	}
}

// TODO: fix "Fragile flower (start)//output_log.txt"
func ReceiveFilesWs(conn net.Conn, username string) {
	defer conn.Close()
	if Settings.RootFolder[len(Settings.RootFolder)-1] != '/' {
		Settings.RootFolder += "/"
	}

	for {
		err := wsutil.WriteServerText(conn, []byte("next"))
		utils.CheckError(err, "api.uploadFilesWs() [1]", false)

		msg, err := wsutil.ReadClientText(conn)
		utils.CheckError(err, "api.uploadFilesWs() [2]", false)
		if string(msg) == "stop###" {
			break
		}
		dir := Settings.RootFolder + username + "/" + string(msg)
		if len(string(msg)) != 0 {
			dir += "/"
		}

		msg, err = wsutil.ReadClientText(conn)
		utils.CheckError(err, "api.uploadFilesWs() [3]", false)
		lenFiles, err2 := strconv.Atoi(string(msg))
		utils.CheckError(err2, "api.uploadFilesWs() [4]", false)

		for i := 0; i < lenFiles; i++ {
			err = wsutil.WriteServerText(conn, []byte("next"))
			utils.CheckError(err, "api.uploadFilesWs() [5]", false)

			msg, _, err = wsutil.ReadClientData(conn)
			utils.CheckError(err, "api.uploadFilesWs() [6]", false)
			file := dir + string(msg)

			if ok, err := Exists(dir); !ok {
				err2 = CreateDir(dir)
				utils.CheckError(err, "api.uploadFilesWs() [7]", false)
			} else {
				utils.CheckError(err, "api.uploadFilesWs() [8]", false)
			}

			ReceiveFile(conn, file, "buff")
			// go GetFileHash(file)
		}
	}

	fmt.Println("All files received!")
}

func receiveInt64(reader fileReader) (result int64) {
	buff := make([]byte, 8)
	_, err := io.ReadFull(reader, buff)
	utils.CheckError(err, "receiveInt64 [1]", false)
	result = int64(binary.BigEndian.Uint64(buff))
	utils.CheckError(err, "receiveInt64 [2]", false)
	return
}

func ReceiveFile(conn net.Conn, dstFile string, connectionType string) {
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
	fmt.Printf("%s md5 checksum: %x\n", filename, sum)
}
