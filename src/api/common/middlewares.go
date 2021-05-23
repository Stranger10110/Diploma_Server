package common

import (
	"../../utils"
	"encoding/json"
	"fmt"
	"github.com/adam-hanna/custom_jwt-auth/jwt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"time"

	"github.com/unrolled/secure"
	permissions "github.com/xyproto/custom_permissions2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// TODO: replace method or rename fields
type Credentials struct {
	Cid       string `json:"clientid"  validate:"required"`
	Secret    string `json:"secret"  validate:"required"`
	Mysql     string `json:"mysql"` // TODO: remove
	RedisHost string `json:"redis_host"  validate:"required"`
	RedisPass string `json:"redis_pass"  validate:"required"`
}

var cred Credentials

var Secure *secure.Secure

var JwtAuth jwt.Auth

var Permissions *permissions.Permissions
var UserStates *permissions.UserState

func ReadCredFile(filePath string, data *Credentials) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("File error: %v\n", err)
		os.Exit(1)
	}
	err = json.Unmarshal(file, data)
	utils.CheckError(err, "api middlewares ReadCredFile [1]", false)
}

func init() {
	var err error
	ReadCredFile("./settings/creds.json", &cred) // TODO: move file somewhere else

	err = validator.New().Struct(cred)
	utils.CheckError(err, "api middlewares init [1]", false)

	// ~ Secure middleware ~
	Secure = secure.New(secure.Options{
		// ContentTypeNosniff: true,
		// FrameDeny:          true,
		SSLRedirect:  false, // TODO: enable
		SSLForceHost: false, // TODO: enable
	})

	// ~ JWT middleware ~
	err = jwt.New(&JwtAuth, jwt.Options{
		SigningMethodString:   "RS256",
		PrivateKeyLocation:    "keys/jwt.rsa",     // `$ openssl genrsa -out app.rsa 2048`
		PublicKeyLocation:     "keys/jwt.rsa.pub", // `$ openssl rsa -in app.rsa -pubout > app.rsa.pub`
		RefreshTokenValidTime: 72 * time.Hour,     // TODO: add a config
		AuthTokenValidTime:    15 * time.Minute,
		BearerTokens:          false,
		Debug:                 false,
		IsDevEnv:              true, // TODO: change
	})
	utils.CheckError(err, "api middlewares.init() jwt", false)

	// ~ Permissions middleware ~
	UserStates, err = permissions.NewUserStateWithPassword2(cred.RedisHost, cred.RedisPass, 10) // TODO: test max
	utils.CheckError(err, "api middlewares.init() UserStates [1]", false)
	err = UserStates.SetPasswordAlgo("bcrypt")
	utils.CheckError(err, "api middlewares.init() UserStates [2]", false)
	Permissions = permissions.NewPermissions(UserStates)
	utils.CheckError(err, "api middlewares.init() UserStates [3]", false)
	//// Blank slate, no default permissions
	//Permissions.Clear()
	//Permissions.SetPublicPath([]string{"/login", "/register", "/favicon.ico", "/img",
	//	"/js", "/robots.txt", "/sitemap_index.xml"})
	//Permissions.SetUserPath([]string{"/api"})
	//Permissions.SetAdminPath([]string{"/api/admin", "/secret123_admin"})

	// Add account for the internal use
	if !(UserStates.HasUser("CloudServerData")) {
		UserStates.AddUser("CloudServerData", "sao40jr127dKFNM65d123okabAKXNQ99", "no")
	}

	// Set initial admin accounts
	UserStates.SetAdminStatus("CloudServerData")

	// TODO: remove
	UserStates.SetAdminStatus("test2")
	err = UserStates.SetKey("test2", "grp_test_group", "") // set group
	utils.CheckError(err, "api middlewares.init() UserStates [4]", false)

	// user account keys (all values are strings; _ is empty string)
	// "Recent_del": [_, 1] - if there was a recent delete (it's needed for uploads, because Fuse is kinda slow about delete updates)
}

func SecureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := Secure.Process(c.Writer, c.Request)

		// If there was an error, do not continue.
		if err != nil {
			c.Abort()
			return
		}

		// Avoid header rewrite if response is a redirection.
		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}

		c.Next()
	}
}

func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := JwtAuth.Process(c.Writer, c.Request)

		// If there was an error, do not continue.
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			fmt.Fprint(c.Writer, "Permission denied (jwt)!")
			return
		}

		// Set username parameter in context
		claims, err2 := JwtAuth.GrabTokenClaims(c.Request)
		utils.CheckError(err2, "JwtMiddleware()", false)
		c.Set("username", claims.CustomClaims["usrn"])

		c.Next()
	}
}

//func PermissionMiddleware() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// Check if the user has the right admin/user rights
//		// Set up a middleware handler for Gin, with a custom "permission denied" message.
//
//		// Get username parameter from jwt middleware
//		var username string
//		if user, ok := c.Get("username"); ok {
//			username = fmt.Sprintf("%v", user)
//		} else {
//			username = ""
//		}
//
//		if Permissions.Rejected(username, c.Request) {
//			// Deny the request, don't call other middleware handlers
//			c.AbortWithStatus(http.StatusUnauthorized)
//			fmt.Fprint(c.Writer, "Permission denied!")
//			return
//		}
//		// Call the next middleware handler
//		c.Next()
//	}
//}
