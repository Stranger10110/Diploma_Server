package api

import (
	"../utils"
	"encoding/json"
	"io/ioutil"
)

type settings struct {
	WeedBinaryPath string `json:"weed_binary_path"`
}

var Settings settings

func init() {
	file, err := ioutil.ReadFile("./settings/api_endpoints.json")
	utils.CheckError(err, "api.init [1]", false)
	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "api.init [2]", false)
}
