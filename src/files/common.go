package files

import (
	"../utils"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type settings struct {
	FilerRootFolder string `json:"filer_root_folder"`
	FilerTempFolder string `json:"filer_temp_folder"`
}

var Settings settings
var FileMode map[string]int

func init() {
	file, err := ioutil.ReadFile("./settings/files.json")
	utils.CheckError(err, "files.init [1]", false)
	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "files.init [2]", false)

	if Settings.FilerRootFolder[len(Settings.FilerRootFolder)-1] != '/' {
		Settings.FilerRootFolder += "/"
	}
	if Settings.FilerTempFolder[len(Settings.FilerTempFolder)-1] != '/' {
		Settings.FilerTempFolder += "/"
	}

	FileMode = make(map[string]int)
	FileMode["rb"] = os.O_RDONLY
	FileMode["rb+"] = os.O_RDWR
	FileMode["wb"] = os.O_CREATE | os.O_WRONLY
	FileMode["wb+"] = os.O_CREATE | os.O_RDWR
	FileMode["wob"] = os.O_WRONLY
	FileMode["wob+"] = os.O_RDWR
	FileMode["ab"] = os.O_APPEND | os.O_WRONLY
}

func GetFileHash(filename string) {
	// Open file for reading
	file, err := os.Open(filename)
	utils.CheckError(err, "files.GetFileHash [1]", false)
	defer file.Close()

	// Create new hasher, which is a writer interface
	hasher := md5.New()
	_, err = io.Copy(hasher, file)
	utils.CheckError(err, "files.GetFileHash [2]", false)

	// Hash and print. Pass nil since
	// the data is not coming in as a slice argument
	// but is coming through the writer interface
	sum := hasher.Sum(nil)
	fmt.Printf("%s md5 checksum: %x\n", filename, sum)
}

func OSReadDir(root string) ([]os.FileInfo, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return fileInfo, err
	}

	return fileInfo, nil
}

func RemoveContents(dirFullPath string) error {
	d, err := os.Open(dirFullPath)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dirFullPath, name))
		if err != nil {
			return err
		}
	}
	return nil
}
