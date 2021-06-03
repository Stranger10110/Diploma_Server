package files

import (
	"../utils"

	"bytes"
	"fmt"
	"github.com/gobwas/ws/wsutil"
	"net"
	"strconv"
)

func WsReceiveByteMessage(conn net.Conn) []byte {
	msg, err := wsutil.ReadClientText(conn)
	utils.CheckError(err, "api.files.ReceiveMessage() [1]", false)
	return msg
}

func receiveMessage(conn net.Conn) string {
	msg, err := wsutil.ReadClientText(conn)
	utils.CheckError(err, "api.files.receiveMessage() [1]", false)
	return string(msg)
}

func sendReceiveMessage(conn net.Conn) string {
	err := wsutil.WriteServerText(conn, []byte("next"))
	utils.CheckError(err, "api.files.sendReceiveMessage() [1]", false)

	msg, err := wsutil.ReadClientText(conn)
	utils.CheckError(err, "api.files.sendReceiveMessage() [2]", false)
	return string(msg)
}

func sendLastStatus(conn net.Conn) {
	if err := recover(); err == nil {
		err2 := wsutil.WriteServerText(conn, []byte("stop"))
		utils.CheckError(err2, "api.files.sendLastStatus() [1]", false)
	} else {
		fmt.Println(err)
		err2 := wsutil.WriteServerText(conn, []byte("error"))
		utils.CheckError(err2, "api.files.sendLastStatus() [2]", false)
	}
}

// TODO: fix "Fragile flower (start)//output_log.txt" (can't remember if fixed)
func WsReceiveFiles(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)

	for {
		msg := sendReceiveMessage(conn)
		if msg == "stop###" {
			break
		}
		relPath := username + "/" + msg
		dir := Settings.FilerRootFolder + relPath
		if len(msg) != 0 {
			dir += "/"
		}

		msg = receiveMessage(conn)
		lenFiles, err := strconv.Atoi(msg)
		utils.CheckError(err, "api.files.WsReceiveFiles() [1]", false)

		for i := 0; i < lenFiles; i++ {
			msg = sendReceiveMessage(conn)
			file := dir + msg

			err = CreateDirIfNotExists(dir)
			utils.CheckError(err, "api.files.WsReceiveFiles() [2]", false)
			ReceiveFile(conn, file)

			err = GenerateFileSig(relPath)
			utils.CheckError(err, "api.files.WsReceiveFiles() [3]", false)
			// go GetFileMd5Hash(file)
		}
	}

	fmt.Println("All files received!")
}

func WsReceiveFile(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)
	relPath := sendReceiveMessage(conn)
	filePath := Settings.FilerRootFolder + username + "/" + relPath
	ReceiveFile(conn, filePath)
}

func WsReceiveFileIntoMemory(conn net.Conn, username string) (relPath string, data *bytes.Buffer) {
	defer sendLastStatus(conn)
	relPath = sendReceiveMessage(conn)
	relPath = username + "/" + relPath
	data = ReceiveFileIntoMemory(conn)
	return
}
