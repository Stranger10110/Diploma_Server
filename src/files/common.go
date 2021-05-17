package files

import (
	"../utils"
	"crypto/md5"
	"encoding/json"
	"github.com/go-playground/validator"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type settings struct {
	FilerRootFolder string `json:"filer_root_folder" validate:"required"`
	FilerTempFolder string `json:"filer_temp_folder" validate:"required"`
}

var Settings settings
var FileMode map[string]int

func init() {
	file, err := ioutil.ReadFile("./settings/files.json")
	utils.CheckError(err, "files.init [1]", false)

	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "files.init [2]", false)

	err = validator.New().Struct(Settings)
	utils.CheckError(err, "files.init [3]", false)

	if Settings.FilerRootFolder[len(Settings.FilerRootFolder)-1] != '/' {
		Settings.FilerRootFolder += "/"
	}
	if Settings.FilerTempFolder[len(Settings.FilerTempFolder)-1] != '/' {
		Settings.FilerTempFolder += "/"
	}

	err = CreateDirIfNotExists(Settings.FilerTempFolder)
	utils.CheckError(err, "files.init [4]", false)

	FileMode = make(map[string]int)
	FileMode["rb"] = os.O_RDONLY
	FileMode["rb+"] = os.O_RDWR
	FileMode["wb"] = os.O_CREATE | os.O_WRONLY
	FileMode["wb+"] = os.O_CREATE | os.O_RDWR
	FileMode["wob"] = os.O_WRONLY
	FileMode["wob+"] = os.O_RDWR
	FileMode["ab"] = os.O_APPEND | os.O_WRONLY
}

func GetFileMd5Hash(filename string) []byte {
	// Open file for reading
	file, err := os.Open(filename)
	utils.CheckError(err, "files.GetFileMd5Hash [1]", false)
	defer file.Close()

	// Create new hasher, which is a writer interface
	hasher := md5.New()
	_, err = io.Copy(hasher, file)
	utils.CheckError(err, "files.GetFileMd5Hash [2]", false)

	// Hash. Pass nil since
	// the data is not coming in as a slice argument
	// but is coming through the writer interface
	sum := hasher.Sum(nil)
	return sum
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
	err := os.MkdirAll(folderPath, 0600)
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
