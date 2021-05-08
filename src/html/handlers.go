package html

import (
	"../api"
	"../files"
	"../utils"

	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func generateFileHtml(f os.FileInfo) string {
	template := "<tr><td><div class=\"file link-alike\" onclick=\"[function]\">[filename]</div></td><td>[size]</td><td>[date]</td></tr>\n"
	html := ""
	var size, name, function string

	if f.Size() == 0 {
		size = ""
	} else {
		size = utils.BetterFloatFormat(float64(f.Size())/1024.0) + "MB"
	}

	if f.IsDir() {
		name = f.Name() + "/"
		function = "return folderClicked(this);"
	} else {
		name = f.Name()
		function = "return fileClicked(this);"
	}

	html += strings.NewReplacer(
		"[function]", function,
		"[filename]", name,
		"[size]", size,
		"[date]", f.ModTime().Format("02.01.2006 15:04"),
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
