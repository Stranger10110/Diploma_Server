package files

import (
	"../filer"
	"../rdiff"
	"../utils"

	"golang.org/x/sys/unix"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

func WsMakeVersionDelta(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)

	// Receive relative file path and signature of "basis" file (of a new file version to make delta to make old version)
	FileRelPath, sigFileData := WsReceiveFileIntoMemory(conn, username)
	sigFd, _, err := MemFile(FileRelPath, sigFileData.Bytes(), "rb")
	utils.CheckError(err, "files.WsMakeVersionDelta [1]", false)
	defer unix.Close(sigFd)

	filePath := Settings.FilerRootFolder + FileRelPath
	metaPath := Settings.FilerRootFolder + "Meta_" + filepath.Dir(FileRelPath)
	CreateDirIfNotExists(metaPath)

	// Get file current version and make +1
	sigPath_ := metaPath + "/" + filepath.Base(FileRelPath) + ".sig.v"
	var currentFileVersion, newFileVersion string
	sig, _ := filepath.Glob(sigPath_ + "*")
	if sig != nil {
		version, err2 := strconv.Atoi(filepath.Ext(sig[0])[2:])
		utils.CheckError(err2, "files.WsMakeVersionDelta [2]", false)
		currentFileVersion = strconv.Itoa(version)
		newFileVersion = strconv.Itoa(version + 1)
	}

	deltaFile := metaPath + "/" + filepath.Base(FileRelPath) + ".delta.v" + currentFileVersion
	CreateFileIfNotExists(deltaFile)

	res := rdiff.Rdiff.Delta2(sigFd, filePath, deltaFile, "wb")
	if res != 0 {
		err = os.Remove(deltaFile)
		utils.CheckError(err, "files WsMakeVersionDelta [3]", false)
	}

	defer func() { // Remove delta and new signature if any errors ahead
		if err3 := recover(); err3 != nil {
			err = os.Remove(deltaFile)
			utils.CheckError(err, "files WsMakeVersionDelta [4]", false)

			err = os.Remove(sigPath_ + newFileVersion)
			utils.CheckError(err, "files WsMakeVersionDelta [5]", false)

			filer.RemoveFileLock(FileRelPath)
		}
	}()
	err = ioutil.WriteFile(sigPath_+newFileVersion, sigFileData.Bytes(), 0600)
	utils.CheckError(err, "files.WsMakeVersionDelta [6]", false)

	err = os.Rename(sigPath_+currentFileVersion, sigPath_)
	utils.CheckError(err, "files.WsMakeVersionDelta [7]", false)

	filer.RemoveFileLock(FileRelPath)
}

func WsReceiveDelta(conn net.Conn, username string) (deltaPath string, relPath string) {
	defer sendLastStatus(conn)

	relPath = sendReceiveMessage(conn)
	relPath = username + "/" + relPath
	deltaPath = Settings.FilerRootFolder + "Meta_" + relPath + ".delta_new"
	ReceiveFile(conn, deltaPath)
	return
}

func WsReceiveNewFileVersion(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)

	// Receive delta
	deltaPath, FileRelPath := WsReceiveDelta(conn, username)
	// Apply delta and make "[filename]_2"
	filePath := Settings.FilerRootFolder + FileRelPath
	res := rdiff.Rdiff.Patch(filePath, deltaPath, filePath+"_2", "wb")
	if res != 0 {
		err := os.Remove(deltaPath)
		utils.CheckError(err, "files WsReceiveNewFileVersion [1]", false)
	}

	// Remove delta and new file if any errors ahead
	defer func() {
		if err3 := recover(); err3 != nil {
			err := os.Remove(deltaPath)
			utils.CheckError(err, "files WsReceiveNewFileVersion [2]", false)

			err = os.Remove(filePath + "_2")
			utils.CheckError(err, "files WsReceiveNewFileVersion [3]", false)

			filer.RemoveFileLock(FileRelPath)
		}
	}()

	// Remove file old version
	err := os.Remove(filePath)
	utils.CheckError(err, "files WsReceiveNewFileVersion [4]", false)
	// Rename "[filename]_2" to "[filename]"
	err = os.Rename(filePath+"_2", filePath)
	utils.CheckError(err, "files.WsReceiveNewFileVersion [5]", false)

	// Remove "*.sig.v" file
	err = os.Remove(Settings.FilerRootFolder + "Meta_" + FileRelPath + ".sig.v")
	utils.CheckError(err, "files WsReceiveNewFileVersion [6]", false)
	// Remove new delta
	err = os.Remove(deltaPath)
	utils.CheckError(err, "files WsReceiveNewFileVersion [7]", false)
	// Remove file lock
	filer.RemoveFileLock(FileRelPath)
}

func WsSendNewFileVersion(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)

	// Receive relative file path and signature
	FileRelPath, sigFileData := WsReceiveFileIntoMemory(conn, username)
	sigFd, _, err := MemFile(FileRelPath, sigFileData.Bytes(), "rb")
	utils.CheckError(err, "files.WsSendNewFileVersion [1]", false)
	defer unix.Close(sigFd)

	// Make and send delta at the same time
	if receiveMessage(conn) != "next" {
		return
	}

	filePath := Settings.FilerRootFolder + FileRelPath
	tcpFile := getFileFromTcp(conn)
	defer tcpFile.Close()

	res := rdiff.Rdiff.Delta3(sigFd, filePath, int(tcpFile.Fd()), "wob")
	if res != 0 {
		panic("files WsSendNewFileVersion [2]: could not make/send delta!")
	}
	// //

	// Remove file lock
	filer.RemoveFileLock(FileRelPath)
}
