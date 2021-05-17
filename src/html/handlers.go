package html

import (
	"../api"
	"../files"
	"../utils"
	"fmt"

	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func generateFileHtml(f os.FileInfo) string {
	template := "<tr><td><div class=\"with-delete\"> <div style=\"display:inline-flex; align-items:center;\">[folder]<div class=\"file link-alike\" onclick=\"[function]\">[filename]</div></div> <i class=\"far fa-trash-alt delete-btn\" onclick=\"deleteClicked(this);\"></i> </div></td>  <td>[size]</td><td>[date]</td></tr>\n"
	html := ""
	var folder, size, function string

	if f.IsDir() || f.Size() == 0 {
		size = ""
	} else {
		size = fmt.Sprintf("%.2f", float64(f.Size())/1048576.0) + "MB"
		if size == "0.00MB" {
			size = fmt.Sprintf("%.2f", float64(f.Size())/1024.0) + "KB"
		}
	}

	if f.IsDir() {
		function = "folderClicked(this);"
		folder = "<i class=\"far fa-folder\" style=\"margin-right: 4px;\"></i>" // padding-top: 2px;
	} else {
		function = "downloadFile(this);"
		folder = ""
	}

	html += strings.NewReplacer(
		"[folder]", folder,
		"[function]", function,
		"[filename]", f.Name(),
		"[size]", size,
		"[date]", f.ModTime().Format("02.01.2006, 15:04"),
	).Replace(template)
	return html
}

func generateFolderListing(folderPath string) string {
	dirList, _ := files.OSReadDir(folderPath)
	html := ""
	for _, f := range dirList {
		html += generateFileHtml(f)
	}
	return html
}

func generatePath(folderPath string) (html string) {
	html = "<div class=\"link-alike path-part\" id=\"path-root\" onclick=\"folderClicked('')\"> ./ </div>\n"
	if folderPath == "/" {
		return strings.Replace(html, "''", "'#'", 1)
	}

	template := "<div class=\"link-alike path-part\" onclick=\"folderClicked('[increment]')\"> [path] </div>\n"
	paths := strings.Split(folderPath, "/")
	pathsLength := len(paths)
	var incrementPath string
	for _, path := range paths[:pathsLength-2] {
		if path != "" {
			path += "/"
			incrementPath += path
			html += strings.NewReplacer("[increment]", incrementPath, "[path]", path).Replace(template)
		}
	}

	html += strings.NewReplacer("[increment]", "#", "[path]", paths[pathsLength-2]+"/").Replace(template)
	return
}

func FilerListing(c *gin.Context) {
	username := api.GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	path := files.Settings.FilerRootFolder + username + c.Param("reqPath")
	fileInfo, err := os.Stat(path)
	if utils.CheckErrorForWeb(err, "html FilerListing [1]", c) {
		return
	}

	switch mode := fileInfo.Mode(); {
	case mode.IsRegular():
		c.AbortWithStatus(http.StatusBadRequest)
	case mode.IsDir():
		c.String(http.StatusOK, generateFolderListing(path)+"^^^"+generatePath(c.Param("reqPath")))
	}
}
