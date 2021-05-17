package main_settings

import (
	"../utils"
	"encoding/json"
	"github.com/go-playground/validator"
	"io/ioutil"
)

type settings struct {
	FilerAddress    string `json:"filer_address"  validate:"required"`
	MasterAddresses string `json:"master_addresses"  validate:"required"`
	Method          string `json:"method"  validate:"required"`
}

var Settings settings

func init() {
	file, err := ioutil.ReadFile("./settings/main.json")
	utils.CheckError(err, "main_settings.init [1]", false)

	Settings = settings{}
	err = json.Unmarshal([]byte(file), &Settings)
	utils.CheckError(err, "main_settings.init [2]", false)

	err = validator.New().Struct(Settings)
	utils.CheckError(err, "main_settings.init [3]", false)

	if Settings.FilerAddress[len(Settings.FilerAddress)-1] != '/' {
		Settings.FilerAddress += "/"
	}
}
