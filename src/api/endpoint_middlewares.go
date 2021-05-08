package api

import (
	"../utils"
	apiCommon "./common"
	jsonLib "encoding/json"

	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GET /public_share/:link, /secure_share/:link middleware
func GetPathFromLink(c *gin.Context) {
	hash := c.Param("link")
	linkType := hash[len(hash)-1]
	hash = hash[:len(hash)-1]

	//err2 := apiCommon.UserStates.DelKey("CloudServerData", "h_df6f97b963c9b3c8ef00e0c927315aaf58c58183f48082625ebb790aac90b19e")
	//utils.CheckErrorForWeb(err2, "bas", c)

	var path, key, serverLinkTypeShouldBe string

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
		serverLinkType := split[2]
		if !strings.HasPrefix(serverLinkType, serverLinkTypeShouldBe) {
			if utils.CheckErrorForWeb(err, "api endpoints GetPathFromLink [1]", c) {
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Check expiration time
		expTime, err2 := strconv.ParseInt(split[1], 10, 64)
		if err2 != nil {
			if utils.CheckErrorForWeb(err2, "api endpoints GetPathFromLink [2]", c) {
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
			username := getUserName(c)
			if username == "" {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			group := "grp_" + serverLinkType[2:]
			if has, err3 := apiCommon.UserStates.HasKey(username, group); err3 == nil && !has {
				c.AbortWithStatus(http.StatusForbidden)
				return
			} else if err3 != nil {
				if utils.CheckErrorForWeb(err2, "api endpoints GetPathFromLink [3]", c) {
					return
				}
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		// Set path if everything before is ok
		path = split[0]
	} else {
		if utils.CheckErrorForWeb(err, "api endpoints GetPathFromLink [4]", c) {
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set("Proxy_path", path)
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
	username := getUserName(c)
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