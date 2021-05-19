package filer

import (
	s "../main_settings"
	"../utils"
	"net/http"
	"strings"
)

var client *http.Client
var Uuid string

func init() {
	client = &http.Client{}
	Uuid = "Cloud_storage_" + utils.TokenGenerator(2)
}

// TODO: add error checking and repeat if needed (where these functions are used)

func SetFileLock(FullRelPath string) *http.Response {
	req, err := http.NewRequest(http.MethodPut, "http://"+s.Settings.FilerAddress+FullRelPath+"?tagging", nil)
	utils.CheckError(err, "filer.SetFileLock() [1]", true)

	req.Header.Set("Seaweed-Lock", Uuid)
	resp, err := client.Do(req)
	utils.CheckError(err, "filer.SetFileLock() [2]", true)
	return resp
}

func GetFileLock(FullRelPath string) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodHead, "http://"+s.Settings.FilerAddress+FullRelPath+"?tagging", nil)
	utils.CheckError(err, "filer.GetFileLock() [1]", true)

	resp, err := client.Do(req)
	utils.CheckError(err, "filer.GetFileLock() [2]", true)
	if resp != nil {
		return resp, resp.Header.Get("Seaweed-Lock")
	} else {
		return resp, ""
	}
}

func RemoveFileLock(FullRelPath string) *http.Response {
	req, err := http.NewRequest(http.MethodDelete,
		"http://"+s.Settings.FilerAddress+FullRelPath+"?tagging=Lock", nil)
	utils.CheckError(err, "filer.RemoveFileLock() [1]", true)

	resp, err := client.Do(req)
	utils.CheckError(err, "filer.RemoveFileLock() [2]", true)
	return resp
}

func RemoveFileTags(FullRelPath string, tagNames []string) *http.Response {
	req, err := http.NewRequest(http.MethodDelete,
		"http://"+s.Settings.FilerAddress+FullRelPath+"?tagging="+strings.Join(tagNames, ","), nil)
	utils.CheckError(err, "filer.RemoveFileLock() [1]", true)

	resp, err := client.Do(req)
	utils.CheckError(err, "filer.RemoveFileLock() [2]", true)
	return resp
}
