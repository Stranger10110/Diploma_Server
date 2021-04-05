package files

import (
	"../utils"
	"encoding/json"
	"io/ioutil"
)

type settings struct {
	RootFolder string
}

var Settings settings

func init() {
	file, err := ioutil.ReadFile("./settings/files.json")
	utils.CheckError(err, "files.init [1]", false)
	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "files.init [2]", false)
}
