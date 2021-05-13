package files

import (
	"../filer"
	"../rdiff"
	"../utils"
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func GenerateFileSig(relPath string) error {
	filePath := Settings.FilerRootFolder + relPath
	// Wait for Filer FUZE mount update
	for {
		ok, err := Exists(filePath)
		utils.CheckError(err, "api.files.GenerateFileSig() [1]", false)
		if ok {
			break
		}
		time.Sleep(35 * time.Millisecond)
	}

	// sigPath := Filepath.Dir(filepath) + "/" + Filepath.Base(filepath) + ".sig.v1"
	sigPath := Settings.FilerRootFolder + "Meta_" + relPath + ".sig.v1"
	CreateDirIfNotExists(filepath.Dir(sigPath))
	res := rdiff.Rdiff.Signature(filePath, sigPath, "wb")
	if res == 100 { // RS_IO_ERROR
		// return errors.New("rdiff.Signature error " + sigPath)
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Signature(filePath, sigPath, "wb")
			if res == 0 {
				break
			} else if count == 15 {
				return errors.New("GenerateFileSig rdiff error " + sigPath)
			}
			count += 1
		}
	}
	return nil
}

func WsMakeVersionDelta(conn net.Conn, username string) {
	defer conn.Close()
	defer sendLastStatus(conn)

	// Receive relative file path and signature of "basis" file (of a new file version to make delta to make old version)
	fileRelPath, sigFileData := WsReceiveFileIntoMemory(conn, username)
	sigFd, _, err := MemFile(fileRelPath, sigFileData.Bytes(), "rb")
	utils.CheckError(err, "files.WsMakeVersionDelta [1]", false)
	defer unix.Close(sigFd)

	filePath := Settings.FilerRootFolder + fileRelPath
	metaFilePath_ := Settings.FilerRootFolder + "Meta_" + fileRelPath
	CreateDirIfNotExists(filepath.Dir(metaFilePath_))

	// Get file current version and make +1
	sigPath_ := metaFilePath_ + ".sig.v"
	sig, _ := filepath.Glob(sigPath_ + "*")
	version, err2 := strconv.Atoi(filepath.Ext(sig[0])[2:]) // removed "if sig != nil" // TODO: test
	utils.CheckError(err2, "files.WsMakeVersionDelta [2]", false)
	currentFileVersion := strconv.Itoa(version)
	newFileVersion := strconv.Itoa(version + 1)

	deltaPath := metaFilePath_ + ".delta.v" + currentFileVersion
	res := rdiff.Rdiff.Delta2(sigFd, filePath, deltaPath, "wb")
	if res == 100 { // RS_IO_ERROR
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Delta2(sigFd, filePath, deltaPath, "wb")
			if res == 0 {
				break
			} else if count == 15 {
				err = os.Remove(deltaPath)
				utils.CheckError(err, "files WsMakeVersionDelta [3]", false)
			}
			count += 1
		}
	} else if res != 0 {
		err = os.Remove(deltaPath)
		utils.CheckError(err, "files WsMakeVersionDelta [3]", false)
	}

	// Remove delta and new signature if any errors ahead
	defer func() {
		if err3 := recover(); err3 != nil {
			err = os.Remove(deltaPath)
			utils.CheckError(err, "files WsMakeVersionDelta [4]", false)

			err = os.Remove(sigPath_ + newFileVersion)
			utils.CheckError(err, "files WsMakeVersionDelta [5]", false)

			filer.RemoveFileLock(fileRelPath)
		}
	}()

	err = ioutil.WriteFile(sigPath_+newFileVersion, sigFileData.Bytes(), 0600)
	utils.CheckError(err, "files.WsMakeVersionDelta [6]", false)

	err = os.Rename(sigPath_+currentFileVersion, sigPath_)
	utils.CheckError(err, "files.WsMakeVersionDelta [7]", false)

	filer.RemoveFileLock(fileRelPath)
}

func WsReceiveDelta(conn net.Conn, username string) (deltaPath string, relPath string) {
	defer sendLastStatus(conn)

	relPath = sendReceiveMessage(conn)
	relPath = username + "/" + relPath
	deltaPath = Settings.FilerTempFolder + "Meta_" + relPath + ".delta_new"
	CreateDirIfNotExists(filepath.Dir(deltaPath))

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
	if res == 100 { // RS_IO_ERROR
		// return errors.New("rdiff.Signature error " + sigPath)
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Patch(filePath, deltaPath, filePath+"_2", "wb")
			if res == 0 {
				break
			} else if count == 15 {
				err := os.Remove(deltaPath)
				utils.CheckError(err, "files WsReceiveNewFileVersion [1]", false)
			}
			count += 1
		}
	} else if res != 0 {
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

func MakeVersionDelta(newFilePath string, oldFilePath string, currentVersion int,
	currentVersionString string, metaFilePath string, sigPath_ string) error {
	newSigPath := sigPath_ + strconv.Itoa(currentVersion+1)
	res := rdiff.Rdiff.Signature(newFilePath, newSigPath, "wb")
	if res == 100 { // RS_IO_ERROR
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Signature(newFilePath, newSigPath, "wb")
			if res == 0 {
				break
			} else if count == 15 {
				return errors.New("MakeVersionDelta signature error")
			}
			count += 1
		}
	} else if res != 0 {
		return errors.New("MakeVersionDelta signature error")
	}

	// Make delta of old version
	deltaPath := metaFilePath + ".delta.v" + currentVersionString
	res = rdiff.Rdiff.Delta(newSigPath, oldFilePath, deltaPath, "wb")
	if res == 100 { // RS_IO_ERROR
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Delta(newSigPath, oldFilePath, deltaPath, "wb")
			if res == 0 {
				break
			} else if count == 15 {
				err := os.Remove(deltaPath)
				if err != nil {
					return err
				}
				err = os.Remove(newSigPath)
				if err != nil {
					return err
				}
				return errors.New("MakeVersionDelta couldn't create delta")
			}
			count += 1
		}
	} else if res != 0 {
		err := os.Remove(deltaPath)
		if err != nil {
			return err
		}
		err = os.Remove(newSigPath)
		if err != nil {
			return err
		}
		return errors.New("MakeVersionDelta couldn't create delta")
	}

	err := os.Remove(sigPath_ + currentVersionString)
	if err != nil {
		err2 := os.Remove(deltaPath)
		if err2 != nil {
			return err2
		}

		err2 = os.Remove(newSigPath)
		if err2 != nil {
			return err2
		}

		return err
	}
	return nil
}

func DowngradeFileToVersion(downgradeTo int, fileRelPath string, c *gin.Context) {
	// Get file current version number
	metaFilePath := Settings.FilerRootFolder + "Meta_" + fileRelPath
	sigPath_ := metaFilePath + ".sig.v"
	sig, _ := filepath.Glob(sigPath_ + "*")
	currentFileVersionString := filepath.Ext(sig[0])[2:]
	currentFileVersion, err := strconv.Atoi(currentFileVersionString)
	if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [1]", c) {
		return
	}

	if downgradeTo < 0 {
		sub := currentFileVersion + downgradeTo
		if sub > 0 {
			downgradeTo = sub
		} else {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
	} else if downgradeTo >= currentFileVersion {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// While current != downgradeTo
	//// Apply delta №[current - 1] (save copy into temp dir)
	//// current--
	deltaPath_ := metaFilePath + ".delta.v"
	filePath := Settings.FilerRootFolder + fileRelPath
	tempCopyPath1 := filePath
	tempCopyPath2 := Settings.FilerTempFolder + fileRelPath
	CreateDirIfNotExists(filepath.Dir(tempCopyPath2))

	// // First run
	res := rdiff.Rdiff.Patch(tempCopyPath1, deltaPath_+strconv.Itoa(currentFileVersion-1), tempCopyPath2, "wb")
	if res == 100 { // RS_IO_ERROR
		// return errors.New("rdiff.Signature error " + sigPath)
		count := 0
		for {
			time.Sleep(200 * time.Millisecond)
			res = rdiff.Rdiff.Patch(tempCopyPath1, deltaPath_+strconv.Itoa(currentFileVersion-1), tempCopyPath2, "wb")
			if res == 0 {
				break
			} else if count == 15 {
				err = os.Remove(tempCopyPath2)
				if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [2]", c) {
					return
				}
			}
			count += 1
		}
	} else if res != 0 {
		err = os.Remove(tempCopyPath2)
		if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [2]", c) {
			return
		}
	}
	tempCopyPath1 = tempCopyPath2
	tempCopyPath2 += "_2"

	currentVersionCopy := currentFileVersion - 1
	for {
		if currentVersionCopy == downgradeTo {
			break
		}

		res = rdiff.Rdiff.Patch(tempCopyPath1, deltaPath_+strconv.Itoa(currentVersionCopy-1), tempCopyPath2, "wb")
		if res != 0 {
			err = os.Remove(tempCopyPath2)
			if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [3]", c) {
				return
			}
		}

		err = os.Remove(tempCopyPath1)
		if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [4]", c) {
			return
		}

		err = os.Rename(tempCopyPath2, tempCopyPath1)
		if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [5]", c) {
			err = os.Remove(tempCopyPath2)
			if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [6]", c) {
				return
			}
			return
		}

		currentVersionCopy--
	}
	//

	// Make new file version
	err = MakeVersionDelta(tempCopyPath1, filePath, currentFileVersion, currentFileVersionString, metaFilePath, sigPath_)
	if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [7]", c) {
		return
	}

	// Set file lock, move copy into filer, remove lock
repeat:
	if _, lockValue := filer.GetFileLock(fileRelPath); lockValue == "" || lockValue == filer.Uuid {
		filer.SetFileLock(fileRelPath)
		if _, lockValue = filer.GetFileLock(fileRelPath); lockValue == filer.Uuid {
			err = exec.Command("mv", tempCopyPath1, filePath).Run()
			if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [8]", c) {
				err = os.Remove(tempCopyPath1)
				if utils.CheckErrorForWeb(err, "files.DowngradeFileToVersion [9]", c) {
					filer.RemoveFileLock(fileRelPath)
					return
				}
				filer.RemoveFileLock(fileRelPath)
				return
			}
			filer.RemoveFileLock(fileRelPath)
		} else {
			time.Sleep(3 * time.Second)
			goto repeat
		}
	} else {
		time.Sleep(3 * time.Second)
		goto repeat
	}
}

func RemoveFileMetadata(fileRelPath string) (err error) {
	metaFilePath_ := Settings.FilerRootFolder + "Meta_" + fileRelPath
	signatures, _ := filepath.Glob(metaFilePath_ + ".sig.v*")
	deltas, _ := filepath.Glob(metaFilePath_ + ".delta.v*")

	for _, sig := range signatures {
		err = os.Remove(sig)
		if err != nil {
			return err
		}
	}

	for _, delta := range deltas {
		err = os.Remove(delta)
		if err != nil {
			return err
		}
	}
	return nil
}
