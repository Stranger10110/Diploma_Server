package api

import (
	filesApi "../files"
	"../utils"
	apiCommon "./common"
	jsonLib "encoding/json"

	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func changeParam(c *gin.Context, key string, value string) {
	for i, param := range c.Params {
		if param.Key == key {
			c.Params[i].Value = "/" + value
		}
	}
}

// GET /api/shared/link/*reqPath
// PUT, DELETE /api/shared/link
//
// GET /shared/content/:link/*reqPath, /secure/shared/content/:link/*reqPath
//
// GET /api/public/shared/zip/:link/*reqPath, /api/shared/zip/:link/*reqPath
//
// GET, POST, PUT, DELETE  /api/public/shared/filer/:link/*reqPath middleware, /api/shared/filer/:link/*reqPath
// middleware
func SetInfoFromLink(c *gin.Context) {
	hash := c.Param("link")
	linkType := hash[len(hash)-1]
	if (c.Request.RequestURI[0:7] == "/shared") && (linkType == 'b') {
		c.Redirect(http.StatusTemporaryRedirect, "/secure/shared/content/"+hash+c.Param("reqPath"))
		c.Abort()
		return
	} else if (c.Request.RequestURI[0:11] == "/api/public") && linkType == 'b' {
		c.Redirect(http.StatusTemporaryRedirect, "/api/shared/filer/"+hash+c.Param("reqPath"))
		c.Abort()
		return
	}

	hash = hash[:len(hash)-1]

	var path, key, serverLinkTypeShouldBe, linkUsername, permission string

	if linkType == 'a' {
		serverLinkTypeShouldBe = "p"
	} else if linkType == 'b' {
		serverLinkTypeShouldBe = "g"
	} else {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	key = "shl_" + hash
	if value, err := apiCommon.UserStates.GetKey("CloudServerData", key); err == nil && len(value) > 0 {
		split := strings.Split(value, ";")

		// Check link type
		serverLinkType := split[4]
		if !strings.HasPrefix(serverLinkType, serverLinkTypeShouldBe) {
			if utils.CheckErrorForWeb(err, "api endpoints SetInfoFromLink [1]", c) {
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Check expiration time
		expTime, err2 := strconv.ParseInt(split[3], 10, 64)
		if err2 != nil {
			if utils.CheckErrorForWeb(err2, "api endpoints SetInfoFromLink [2]", c) {
				return
			}
			c.AbortWithStatus(http.StatusExpectationFailed)
			return
		}
		if (expTime != 0) && (time.Now().Unix() >= expTime) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Check that username is in this group
		if serverLinkTypeShouldBe == "g" {
			username := GetUserName(c)
			if username == "" {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			group := "group_" + serverLinkType[2:]
			if has, err3 := apiCommon.UserStates.HasKey(username, group); err3 == nil && !has {
				c.AbortWithStatus(http.StatusForbidden)
				return
			} else if err3 != nil {
				if utils.CheckErrorForWeb(err2, "api endpoints SetInfoFromLink [3]", c) {
					return
				}
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		// Check permission if it's a request to Filer
		if strings.Contains(c.Request.RequestURI, "/shared/filer/") {
			if c.Request.Method != "GET" && split[2] != "rw" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			} else if c.Request.Method != "GET" && split[2] == "rw" { // ||
				// (c.Request.Method == "GET" && strings.Contains(c.Request.RequestURI, "api/zip/"))  {
				p := filesApi.Settings.FilerRootFolder + split[0] + "/" + split[1]
				fileInfo, err4 := os.Stat(p)
				//if err4 != nil && err4.(*os.PathError).Err == unix.ENOENT { // no such file
				//	c.AbortWithStatus(http.StatusForbidden)
				//	return
				// } else
				if utils.CheckErrorForWeb(err4, "api endpoints SetInfoFromLink [4]", c) {
					return
				}

				if c.Request.Method == "PUT" && !fileInfo.IsDir() { // ||
					// (c.Request.Method == "GET" && strings.Contains(c.Request.RequestURI, "api/zip/"))) &&
					// !fileInfo.IsDir() { // (method is PUT or 'GET zip') and it's not a directory
					c.AbortWithStatus(http.StatusForbidden)
					return
				} else if !fileInfo.IsDir() { // is a file
					if !strings.Contains(c.Param("reqPath"), filepath.Base(split[1])) { // filenames is not equal
						c.AbortWithStatus(http.StatusForbidden)
						return
					}
				}
			}
		}

		// Set path and username if everything before is ok
		permission = split[2]
		path = split[1]
		linkUsername = split[0]
	} else {
		if err != nil && err.Error() != "redigo: nil returned" &&
			utils.CheckErrorForWeb(err, "api endpoints SetInfoFromLink [5]", c) {
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set("basePath", path)

	reqPath := c.Param("reqPath")
	if reqPath == "/" {
		reqPath = ""
	}
	if !strings.Contains(path, reqPath) {
		path = path + reqPath
	}

	changeParam(c, "reqPath", path)
	c.Set("username", linkUsername)
	c.Set("share_permission", permission)

	c.Next()
}

type filerHeader struct {
	Json string `header:"Fi-js" binding:"required"`
}

type filerUrl struct {
	Path   string            `json:"pat" binding:"required"`
	Params map[string]string `json:"par" binding:"required"`
}

// GET, POST, DELETE /api/filer/ middleware
func GetFilerInfoFromHeader(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var header filerHeader
	if err := c.ShouldBindHeader(&header); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Request.Header.Del("Fi-js")

	var json filerUrl
	if err := jsonLib.Unmarshal([]byte(header.Json), &json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// json.Path = strings.ReplaceAll(json.Path, "\\", "/")
	if _, metaPath := json.Params["meta"]; !metaPath {
		c.Set("Proxy_path", username+"/"+json.Path)
	} else {
		c.Set("Proxy_path", "Meta_"+username+"/"+json.Path)
		delete(json.Params, "meta")
	}

	var params string
	if len(json.Params) > 0 {
		params = "?"
		for param, value := range json.Params {
			params += param
			if value != "" {
				params += "=" + value + "&"
			} else {
				params += "&"
			}
		}
		params = params[:len(params)-1]
	} else {
		params = ""
	}
	c.Set("Proxy_params", params)

	c.Next()
}
