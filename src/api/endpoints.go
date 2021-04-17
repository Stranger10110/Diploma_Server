package api

import (
	filesApi "../files"
	"../utils"
	apiCommon "./common"
	"crypto/md5"
	"fmt"
	jwt_ "github.com/adam-hanna/custom_jwt-auth/jwt"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	// "github.com/cristalhq/jwt"
	"github.com/adam-hanna/jwt-auth/jwt"
	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"net/http"
)

// TODO: add email verification?
func userExists(username string) bool {
	if apiCommon.UserStates.HasUser(username) {
		return true
	}
	return false
}

// TODO: apply rate limiter

type registerStruct struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	// Name string `json:"name" binding:"required"`
	// Surname string `json:"surname" binding:"required"`
}

// POST /api/register
func Register(c *gin.Context) {
	var json registerStruct
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: add email check
	if userExists(json.Username) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	// TODO: add password restrictions
	apiCommon.UserStates.AddUser(json.Username, json.Password, json.Email)

	confirmationCode, err := apiCommon.UserStates.GenerateUniqueConfirmationCode()
	utils.CheckErrorForWeb(err, "apiCommon.Register() - 1", c)
	apiCommon.UserStates.AddUnconfirmed(json.Username, confirmationCode)

	c.JSON(http.StatusOK, gin.H{"status": "registered|confirmation required",
		"code": confirmationCode}) // TODO: send via email
}

type confirmationUser struct {
	Username         string `json:"username" binding:"required"`
	ConfirmationCode string `json:"code" binding:"required"`
}

// POST /api/confirm_username
func ConfirmUser(c *gin.Context) {
	var json confirmationUser
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if code, err := apiCommon.UserStates.ConfirmationCode(json.Username); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	} else if code == json.ConfirmationCode {
		// Confirm user
		if apiCommon.UserStates.HasUser(json.Username) {
			apiCommon.UserStates.Confirm(json.Username)
		} else {
			// TODO: apply rate limiter
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "there is no such user"})
			return
		}

		// Send response
		c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "wrong confirmation code"})
		return
	}
}

type loginUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// POST /api/login
func Login(c *gin.Context) {
	var json loginUser
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	if apiCommon.UserStates.CorrectPassword(json.Username, json.Password) {
		// If unconfirmed user
		if code, _ := apiCommon.UserStates.ConfirmationCode(json.Username); code != "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "confirmation required"})
		}

		claims := jwt.ClaimsType{}
		claims.StandardClaims.Id = utils.TokenGenerator(32)
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["usrn"] = json.Username

		err := apiCommon.JwtAuth.IssueNewTokens(c.Writer, (*jwt_.ClaimsType)(&claims))
		utils.CheckErrorForWeb(err, "api.GetApiKey()", c)

		c.JSON(http.StatusOK, gin.H{"status": "successful"})
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "there is no such user or wrong password"})
	}
}

func getUserName(c *gin.Context) (username string) {
	if user, ok := c.Get("username"); ok {
		username = fmt.Sprintf("%v", user)
	} else {
		username = ""
	}
	return
}

// /api/sync_files
func SyncFilesWs(c *gin.Context) {
	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	utils.CheckErrorForWeb(err, "api.ReceiveFileWs() [1]", c)

	// Get username parameter from jwt middleware
	username := getUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// c.Status(http.StatusAccepted)
	go filesApi.ReceiveFilesWs(conn, username)
}

// TODO: pass filepath as header
func ReverseProxy(address string, handlerBefore string) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote, err := url.Parse(address)
		utils.CheckErrorForWeb(err, "endpoints ReverseProxy [1]", c)

		proxy := httputil.NewSingleHostReverseProxy(remote)
		//Define the director func
		//This is a good place to log, for example
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host

			if handlerBefore == "Filer" {
				req.URL.Path = c.Param("proxyPath")
			} else if handlerBefore == "Share" {
				path, _ := c.Get("shared_link")
				req.URL.Path = "/" + fmt.Sprintf("%v", path)
			}
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// TODO: make method to add username to some group
type SharedFile struct {
	Path    string `json:"path" binding:"required"`     // from root folder (without first slash)
	ExpTime string `json:"exp_time" binding:"required"` // 0 (never) or unix time
	Type    string `json:"type" binding:"required"`     // pub, grp_'name'
}

// GET /api/shared_link
func CreateSharedLink(c *gin.Context) {
	var json SharedFile
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := getUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// apiCommon.UserStates.DelKey("CloudServerData", "h_0c774771090b14b5688d21de6b5a9b85309146454fcff2f365150a20219b89aba")
	path := username + "/" + json.Path
	firstMD5 := md5.Sum([]byte(path))
	hash := fmt.Sprintf("%x", md5.Sum(firstMD5[:]))

	var link string
	var key, value string
	var key2, value2 string

	if json.Type == "pub" {
		// set key for CloudServerData
		key = "shl_" + hash
		value = path + ";" + json.ExpTime + ";p" // last is a link type
		link = hash + "a"                        // public link
	} else if strings.HasPrefix(json.Type, "grp") {
		// set key for CloudServerData
		key = "shl_" + hash
		value = path + ";" + json.ExpTime + ";g_" + json.Type[4:]
		link = hash + "b" // group link
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong link type"})
	}

	// set key for username
	key2 = "shl_" + json.Path
	value2 = hash

	// TODO: check whether file exists on Filer (maybe)
	if has, err := apiCommon.UserStates.HasKey("CloudServerData", key); !has {
		err2 := apiCommon.UserStates.SetKey("CloudServerData", key, value)
		utils.CheckErrorForWeb(err2, "endpoints CreateSharedLinks [1]", c)
	} else {
		utils.CheckErrorForWeb(err, "endpoints CreateSharedLinks [2]", c)
	}

	if has, err := apiCommon.UserStates.HasKey(username, key2); !has {
		err2 := apiCommon.UserStates.SetKey(username, key2, value2)
		utils.CheckErrorForWeb(err2, "endpoints CreateSharedLinks [3]", c)
	} else {
		utils.CheckErrorForWeb(err, "endpoints CreateSharedLinks [4]", c)
	}

	c.JSON(http.StatusOK, gin.H{"link": link})
}

type sFile struct {
	Path     string `json:"path" binding:"required"`
	LinkHash string `json:"link_hash" binding:"required"`
}

// DELETE /api/shared_link
func RemoveSharedLink(c *gin.Context) {
	var json sFile
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := getUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var hashKey, pathKey string
	var err error
	if len(json.Path) > 0 && len(json.LinkHash) > 0 {
		pathKey = "shl_" + json.Path
		hashKey = "shl_" + json.LinkHash[:len(json.LinkHash)-1]

	} else if len(json.Path) > 0 {
		pathKey = "shl_" + json.Path
		hashKey, err = apiCommon.UserStates.GetKey(username, pathKey)
		utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [1]", c)

	} else if len(json.LinkHash) > 0 {
		hashKey = "shl_" + json.LinkHash[:len(json.LinkHash)-1]
		pathKey, err = apiCommon.UserStates.GetKey("CloudServerData", hashKey)
		utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [2]", c)
		pathKey = strings.Split(pathKey, ";")[0]
		pathKey = strings.TrimPrefix(pathKey, username+"/")
	} else {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	err = apiCommon.UserStates.DelKey("CloudServerData", "shl_"+hashKey)
	utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [3]", c)

	err = apiCommon.UserStates.DelKey(username, pathKey)
	utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [4]", c)

	c.Status(http.StatusOK)
}

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
			utils.CheckErrorForWeb(err, "api endpoints GetPathFromLink [1]", c)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Check expiration time
		expTime, err2 := strconv.ParseInt(split[1], 10, 64)
		if err2 != nil {
			utils.CheckErrorForWeb(err2, "api endpoints GetPathFromLink [2]", c)
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
				utils.CheckErrorForWeb(err2, "api endpoints GetPathFromLink [3]", c)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		// Set path if everything before is ok
		path = split[0]
	} else {
		utils.CheckErrorForWeb(err, "api endpoints GetPathFromLink [4]", c)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Set("shared_link", path)
	c.Next()
}
