package html

import (
	"../api"
	"../files"
	"../utils"
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func generateFileHtml(f os.FileInfo) string {
	template := `
		<tr> <td>
			<div class="with-file-actions">
				<div style="display:inline-flex; align-items:center;">
					[icon] <div class="file link-alike" onclick="[function]">[filename]</div>
				</div>

				<div class="file-actions">
					<i class="far fa-share-square hidden-file-btn" onclick="shareClicked(this);"> </i>
					<i class="far fa-trash-alt hidden-file-btn" onclick="deleteClicked(this);"></i>
				</div>
			</div>
		</td>   <td> [size] </td>   <td> [date] </td> </tr>`
	html := ""
	var icon, size, function string

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
		icon = "<i class=\"far fa-folder\"></i>"
	} else {
		function = "downloadFile(this);"
		ext := strings.ToLower(filepath.Ext(f.Name())[1:4])
		switch ext {
		case "doc", "odt":
			icon = "<i class=\"far fa-file-word\"></i>"
		case "pdf":
			icon = "<i class=\"far fa-file-pdf\"></i>"
		case "txt":
			icon = "<i class=\"far fa-file-alt\"></i>"
		case "xls":
			icon = "<i class=\"far fa-file-excel\"></i>"
		case "csv":
			icon = "<i class=\"fas fa-file-csv\"></i>"
		case "ppt":
			icon = "<i class=\"far fa-file-powerpoint\"></i>"
		case "jpg", "jpe", "png", "bmp":
			icon = "<i class=\"far fa-file-image\"></i>"
		}

	}

	html += strings.NewReplacer(
		"[icon]", icon,
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

	var fileList []os.FileInfo
	for _, f := range dirList {
		if f.IsDir() { // Dirs first
			html += generateFileHtml(f)
		} else {
			fileList = append(fileList, f)
		}
	}

	for _, f := range fileList { // Files second
		html += generateFileHtml(f)
	}
	return html
}

func generatePath(folderPath string, basePath string) (html string) {
	folderPath = strings.Replace(folderPath, basePath, "", 1)

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

// GET /filer/*reqPath
func FilerListing(c *gin.Context) {
	username := api.GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	reqPath := c.Param("reqPath")
	path := files.Settings.FilerRootFolder + username + reqPath
	fileInfo, err := os.Stat(path)
	if utils.CheckErrorForWeb(err, "html FilerListing [1]", c) {
		return
	}

	var basePath, permission string
	if strings.Contains(c.Request.RequestURI, "share") {
		x, _ := c.Get("basePath")
		basePath = x.(string)

		x, _ = c.Get("share_permission")
		permission = "^^^" + x.(string)
	} else {
		basePath = ""
		permission = "^^^"
	}

	switch mode := fileInfo.Mode(); {
	case mode.IsRegular():
		c.String(http.StatusOK, generateFileHtml(fileInfo)+"^^^"+generatePath(reqPath, basePath)+permission+"f")
	case mode.IsDir():
		c.String(http.StatusOK, generateFolderListing(path)+"^^^"+generatePath(reqPath, basePath)+permission+"d")
	}
}
