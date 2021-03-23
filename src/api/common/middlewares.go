package common

import (
	// "database/sql"
	"../../utils"
	"encoding/json"
	"fmt"
	"github.com/adam-hanna/jwt-auth/jwt"
	"github.com/gin-gonic/gin"
	"time"

	"github.com/xyproto/custom_permissionsql"
	// "github.com/pjebs/restgate"
	"github.com/unrolled/secure"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// TODO: replace method or rename fields
type Credentials struct {
	Cid    string `json:"clientid"`
	Secret string `json:"secret"`
	Mysql  string `json:"mysql"`
}

var cred Credentials

var Secure *secure.Secure

var JwtAuth jwt.Auth

var Permissions *permissionsql.Permissions
var UserStates *permissionsql.UserState

func ReadCredFile(filePath string, data *Credentials) {
	file, err := ioutil.ReadFile("./creds.json")
	if err != nil {
		log.Printf("File error: %v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(file, data)
}

func init() {
	var err error
	ReadCredFile("./creds.json", &cred) // TODO: move file somewhere else

	// ~ Secure middleware ~
	Secure = secure.New(secure.Options{
		ContentTypeNosniff: true,
		FrameDeny:          true,
		SSLRedirect:        false, // TODO: enable
	})

	// ~ JWT middleware ~
	err = jwt.New(&JwtAuth, jwt.Options{
		// SigningMethodString:   "HS256",
		// HMACKey:			   []byte("S934Sn7_NdBY=B/%m'm@-rgTwUcaw`{}=sz+6K&e.^3u3(V6Dt_7Nk!}aaL7ndPc"),
		SigningMethodString:   "RS256",
		PrivateKeyLocation:    "keys/jwt.rsa",     // `$ openssl genrsa -out app.rsa 2048`
		PublicKeyLocation:     "keys/jwt.rsa.pub", // `$ openssl rsa -in app.rsa -pubout > app.rsa.pub`
		RefreshTokenValidTime: 72 * time.Hour,
		AuthTokenValidTime:    15 * time.Minute,
		BearerTokens:          true,
		Debug:                 true,
		IsDevEnv:              false,
	})
	utils.CheckError(err, "apiCommon.init() jwt", false)

	// ~ Permissions middleware ~
	UserStates, err = permissionsql.NewUserStateWithDSN(cred.Mysql, "UserStates", true)
	utils.CheckError(err, "apiCommon.init() UserStates 1", false)
	err = UserStates.SetPasswordAlgo("bcrypt")
	utils.CheckError(err, "apiCommon.init() UserStates 2", false)
	Permissions = permissionsql.NewPermissions(UserStates) // permissionsql.NewWithDSN(cred.Mysql, "UserStates")
	utils.CheckError(err, "apiCommon.init() UserStates 3", false)
	// Blank slate, no default permissions
	Permissions.Clear()
	Permissions.SetPublicPath([]string{"/login", "/register", "/favicon.ico", "/img",
		"/js", "/robots.txt", "/sitemap_index.xml"})
	Permissions.SetUserPath([]string{"/api"})

	// ~ Restgate middleware ~
	//RG = restgate.New("X-Auth-Key", "X-Auth-Secret",
	//					restgate.Database, restgate.Config{
	//						DB: restgateOpenSqlDb(), TableName: "UserStates",
	//						Key: []string{"keys"}, Secret: []string{"secrets"},
	//					})
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
			c.AbortWithStatus(http.StatusForbidden)
			fmt.Fprint(c.Writer, "Permission denied (jwt)!")
			return
		}

		// Set username parameter for permissions middleware in context
		claims, err2 := JwtAuth.GrabTokenClaims(c.Request)
		utils.CheckError(err2, "JwtMiddleware()", false)
		c.Set("username", claims.CustomClaims["usrn"])

		c.Next()
	}
}

func PermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user has the right admin/user rights
		// Set up a middleware handler for Gin, with a custom "permission denied" message.

		// Get username parameter from jwt middleware
		var username string
		if user, ok := c.Get("username"); ok {
			username = fmt.Sprintf("%v", user)
		} else {
			username = ""
		}

		if Permissions.Rejected(username, c.Request) {
			// Deny the request, don't call other middleware handlers
			c.AbortWithStatus(http.StatusForbidden)
			fmt.Fprint(c.Writer, "Permission denied!")
			return
		}
		// Call the next middleware handler
		c.Next()
	}
}

/*func restgateOpenSqlDb() *sql.DB {
	fmt.Printf("Open mysql: %s\n", cred.Mysql)
	db, err := sql.Open("mysql", cred.Mysql)
	if err != nil {
		return nil
	}

	defer db.Close()
	return db
}

var RG *restgate.RESTGate
func RestgateMiddleware(c *gin.Context) {
	nextCalled := false
	nextAdapter := func(http.ResponseWriter, *http.Request) {
		nextCalled = true
		c.Next()
	}
	RG.ServeHTTP(c.Writer, c.Request, nextAdapter)
	if nextCalled == false {
		c.AbortWithStatus(401)
	}
}*/