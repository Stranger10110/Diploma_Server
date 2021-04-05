package api

import (
	filesApi "../files"
	"../utils"
	apiCommon "./common"
	"fmt"
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
type username struct {
	Username string `json:"username" binding:"required"`
}

// /api/username_exists
func UsernameExists(c *gin.Context) {
	var json username
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if userExists(json.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}
}

type registerStruct struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	// Name string `json:"name" binding:"required"`
	// Surname string `json:"surname" binding:"required"`
}

// /api/register
func Register(c *gin.Context) {
	var json registerStruct
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if userExists(json.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
		return
	}

	// TODO: add password restrictions
	apiCommon.UserStates.AddUser(json.Username, json.Password, json.Email)

	confirmationCode, err := apiCommon.UserStates.GenerateUniqueConfirmationCode()
	utils.CheckError(err, "apiCommon.Register() - 1", false)
	apiCommon.UserStates.AddUnconfirmed(json.Username, confirmationCode)

	c.JSON(http.StatusOK, gin.H{"status": "registered|confirmation required",
		"code": confirmationCode}) // TODO: send via email
}

type confirmationUser struct {
	Username         string `json:"username" binding:"required"`
	ConfirmationCode string `json:"code" binding:"required"`
}

// /api/confirm_username
func ConfirmUser(c *gin.Context) {
	var json confirmationUser
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if code, err := apiCommon.UserStates.ConfirmationCode(json.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	} else if code == json.ConfirmationCode {
		// Confirm user
		if apiCommon.UserStates.HasUser(json.Username) {
			apiCommon.UserStates.Confirm(json.Username)
		} else {
			// TODO: apply rate limiter
			c.JSON(http.StatusBadRequest, gin.H{"error": "there is no such user"})
			return
		}

		// Send response
		c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong confirmation code"})
		return
	}
}

type loginUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// /api/login
func Login(c *gin.Context) {
	var json loginUser
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if apiCommon.UserStates.CorrectPassword(json.Username, json.Password) {
		// If unconfirmed user
		if code, _ := apiCommon.UserStates.ConfirmationCode(json.Username); code != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "confirmation required"})
			return
		}

		claims := jwt.ClaimsType{}
		claims.StandardClaims.Id = utils.TokenGenerator(32)
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["usrn"] = json.Username

		err := apiCommon.JwtAuth.IssueNewTokens(c.Writer, &claims)
		utils.CheckError(err, "api.GetApiKey()", false)

		c.JSON(http.StatusOK, gin.H{"status": "successful"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "there is no such user or wrong password"})
		return
	}
}

// /api/upload_files
func UploadFilesWs(c *gin.Context) {
	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	utils.CheckError(err, "api.ReceiveFileWs() [1]", false)

	// Get username parameter from jwt middleware
	var username string
	if user, ok := c.Get("username"); ok {
		username = fmt.Sprintf("%v", user)
	} else {
		username = ""
	}

	if username == "" {
		c.Status(http.StatusBadRequest)
	}

	// c.Status(http.StatusAccepted)
	go filesApi.ReceiveFilesWs(conn, username)
}
