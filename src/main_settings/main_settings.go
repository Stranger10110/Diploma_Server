package main_settings

import (
	"../utils"
	"encoding/json"
	"io/ioutil"
)

type settings struct {
	FilerAddress    string `json:"filer_address"`
	MasterAddresses string `json:"master_addresses"`
	Method          string `json:"method"`
}

var Settings settings

func init() {
	file, err := ioutil.ReadFile("./settings/main.json")
	utils.CheckError(err, "api.init [1]", false)
	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "api.init [2]", false)
}
