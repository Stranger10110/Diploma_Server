package files

import (
	"../utils"

	"fmt"
	"github.com/gobwas/ws/wsutil"
	"net"
	"strconv"
)

func WsSendReceiveMessage(conn net.Conn) string {
	err := wsutil.WriteServerText(conn, []byte("next"))
	utils.CheckError(err, "api.files.WsSendReceiveMessage() [1]", false)

	msg, err := wsutil.ReadClientText(conn)
	utils.CheckError(err, "api.files.WsSendReceiveMessage() [2]", false)
	return string(msg)
}

func WsReceiveMessage(conn net.Conn) string {
	msg, err := wsutil.ReadClientText(conn)
	utils.CheckError(err, "api.files.WsReceiveMessage() [1]", false)
	return string(msg)
}

// TODO: fix "Fragile flower (start)//output_log.txt" (can't remember if fixed)
func WsReceiveFiles(conn net.Conn, username string) {
	defer conn.Close()

	for {
		msg := WsSendReceiveMessage(conn)
		if msg == "stop###" {
			break
		}
		dir := Settings.FilerRootFolder + username + "/" + msg
		if len(msg) != 0 {
			dir += "/"
		}

		msg = WsReceiveMessage(conn)
		lenFiles, err := strconv.Atoi(msg)
		utils.CheckError(err, "api.files.WsReceiveFiles() [1]", false)

		for i := 0; i < lenFiles; i++ {
			msg = WsSendReceiveMessage(conn)
			file := dir + msg

			if ok, err2 := Exists(dir); err2 == nil && !ok {
				err3 := CreateDir(dir)
				utils.CheckError(err3, "api.files.WsReceiveFiles() [2]", false)
			} else {
				utils.CheckError(err2, "api.files.WsReceiveFiles() [3]", false)
			}

			ReceiveFile(conn, file, "buff")

			err = GenerateFileSigFullPath(file)
			utils.CheckError(err, "api.files.WsReceiveFiles() [4]", false)
			// go GetFileHash(file)
		}
	}

	fmt.Println("All files received!")
}

func WsReceiveFile(conn net.Conn, username string) (filePath string) {
	defer conn.Close()
	err := wsutil.WriteServerText(conn, []byte("next"))
	utils.CheckError(err, "api.files.WsReceiveFile() [1]", false)

	relPath, _, err2 := wsutil.ReadClientData(conn)
	utils.CheckError(err2, "api.files.WsReceiveFile() [2]", false)

	filePath = Settings.FilerRootFolder + username + "/" + string(relPath)
	ReceiveFile(conn, filePath, "buff")

	return filePath
}
