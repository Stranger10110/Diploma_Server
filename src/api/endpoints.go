package api

import (
	"../filer"
	filesApi "../files"
	main "../main_settings"
	"../utils"
	apiCommon "./common"
	"path/filepath"
	"time"

	"bytes"
	"crypto/md5"
	"fmt"
	jwt_ "github.com/adam-hanna/custom_jwt-auth/jwt"
	"github.com/adam-hanna/jwt-auth/jwt"
	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	if utils.CheckErrorForWeb(err, "apiCommon.Register() - 1", c) {
		return
	}
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

		// Set Filer collection // TODO: test
		if has, err := apiCommon.UserStates.HasKey(json.Username, "has_filer_coll"); err == nil && !has {
			cmd := fmt.Sprintf("echo 'fs.configure -locationPrefix=%s -collection=%s -apply' |",
				"/"+json.Username+"/", json.Username)
			cmd += fmt.Sprintf(" %s shell -filer=%s -master=%s", Settings.WeedBinaryPath,
				main.Settings.FilerAddress, main.Settings.MasterAddresses)

			out, err2 := exec.Command("bash", "-c", cmd).Output()
			if utils.CheckErrorForWeb(err2, "api endpoints ConfirmUser [1]", c) {
				return
			}

			fmt.Println(out)
			err2 = apiCommon.UserStates.SetKey(json.Username, "has_filer_coll", "")
			if utils.CheckErrorForWeb(err2, "api endpoints ConfirmUser [2]", c) {
				return
			}
		} else {
			if utils.CheckErrorForWeb(err, "api endpoints ConfirmUser [3]", c) {
				return
			}
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
		claims.StandardClaims.Id = utils.TokenGenerator(16)
		claims.CustomClaims = make(map[string]interface{})
		claims.CustomClaims["usrn"] = json.Username

		err := apiCommon.JwtAuth.IssueNewTokens(c.Writer, (*jwt_.ClaimsType)(&claims))
		if utils.CheckErrorForWeb(err, "api.GetApiKey()", c) {
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "successful"})
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "there is no such user or wrong password"})
	}
}

func GetUserName(c *gin.Context) (username string) {
	if user, ok := c.Get("username"); ok {
		username = fmt.Sprintf("%v", user)
	} else {
		username = ""
	}
	return
}

// TODO: make method to add username to some group
type sharedFile struct {
	Path    string `json:"path" binding:"required"`     // from root folder (without first slash)
	ExpTime string `json:"exp_time" binding:"required"` // 0 (never) or unix time
	Type    string `json:"type" binding:"required"`     // pub, grp_'name'
}

// GET /api/shared_link
func CreateSharedLink(c *gin.Context) {
	var json sharedFile
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := GetUserName(c)
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
		if utils.CheckErrorForWeb(err2, "endpoints CreateSharedLinks [1]", c) {
			return
		}
	} else {
		if utils.CheckErrorForWeb(err, "endpoints CreateSharedLinks [2]", c) {
			return
		}
	}

	if has, err := apiCommon.UserStates.HasKey(username, key2); !has {
		err2 := apiCommon.UserStates.SetKey(username, key2, value2)
		if utils.CheckErrorForWeb(err2, "endpoints CreateSharedLinks [3]", c) {
			return
		}
	} else {
		if utils.CheckErrorForWeb(err, "endpoints CreateSharedLinks [4]", c) {
			return
		}
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

	username := GetUserName(c)
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
		if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [1]", c) {
			return
		}

	} else if len(json.LinkHash) > 0 {
		hashKey = "shl_" + json.LinkHash[:len(json.LinkHash)-1]
		pathKey, err = apiCommon.UserStates.GetKey("CloudServerData", hashKey)
		if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [2]", c) {
			return
		}
		pathKey = strings.Split(pathKey, ";")[0]
		pathKey = strings.TrimPrefix(pathKey, username+"/")
	} else {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	err = apiCommon.UserStates.DelKey("CloudServerData", "shl_"+hashKey)
	if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [3]", c) {
		return
	}

	err = apiCommon.UserStates.DelKey(username, pathKey)
	if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [4]", c) {
		return
	}

	c.Status(http.StatusOK)
}

// Generate file signature after uploading (if statusOK) and before sending response back
func GenerateFileSigFromProxy(relPath string) func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return filesApi.GenerateFileSig(relPath)
		}
		return nil
	}
}

func cleanProxyHeaders(req *http.Request) *http.Request {
	delete(req.Header, "X-Auth-Token")
	delete(req.Header, "X-Csrf-Token")
	delete(req.Header, "X-Refresh-Token")

	if value, ok := req.Header["Accept"]; ok && value[0] == "application/json" {
		delete(req.Header, "Accept-Encoding")
	}

	return req
}

func ReverseProxy(address string, modifyResponse bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		path, _ := c.Get("Proxy_path")
		var params interface{}
		var exists bool
		if params, exists = c.Get("Proxy_params"); !exists {
			params = ""
		}
		remotePath := "/" + fmt.Sprintf("%v", path.(string)+params.(string))
		// fmt.Println(remotePath)

		remote, err := url.Parse(address + remotePath)
		if utils.CheckErrorForWeb(err, "endpoints ReverseProxy [1]", c) {
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		//Define the director func
		//This is a good place to log, for example
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = remote.Path
			req.URL.RawQuery = remote.RawQuery
		}

		if modifyResponse && c.Request.Method == "POST" {
			proxy.ModifyResponse = GenerateFileSigFromProxy(path.(string))
		}

		proxy.ServeHTTP(c.Writer, cleanProxyHeaders(c.Request))
	}
}

func rewriteProxyBody(username string) func(*http.Response) error {
	return func(resp *http.Response) (err error) {
		b, err := ioutil.ReadAll(resp.Body) //Read html
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}

		b = bytes.Replace(b, []byte("<a href=\"/"+username), []byte("<a href=\"/filer"), -1)
		b = bytes.Replace(b, []byte("<a href=\"/\" >\n\t\t\t\t\t /\n\t\t\t\t</a>"), []byte(""), -1)
		b = bytes.Replace(b, []byte(username+" /"), []byte("/"), 1)

		body := ioutil.NopCloser(bytes.NewReader(b))
		resp.Body = body
		resp.ContentLength = int64(len(b))
		resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return nil
	}
}

func changeContentDisposition(resp *http.Response) error {
	resp.Header.Set(
		"Content-Disposition",
		strings.Replace(resp.Header.Get("Content-Disposition"), "inline", "attachment", 1),
	)
	return nil
}

func setParam(c *gin.Context, key string, value string) {
	// Change Param in gin.Context
	for i, param := range c.Params {
		if param.Key == key {
			c.Params[i].Value = value
		}
	}
}

func modifyProxyResponse(proxy *httputil.ReverseProxy, username string, c *gin.Context) {
	if c.Request.Method == "POST" {
		proxy.ModifyResponse = func(resp *http.Response) error {
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				return filesApi.GenerateFileSig(username + c.Param("reqPath"))
			}
			return nil
		}
	}
}

// Reverse proxy for Filer
func ReverseProxy2(address string) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := GetUserName(c)
		if username == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var relPath string
		if strings.Contains(c.Request.URL.RawQuery, "meta") {
			c.Request.URL.RawQuery = strings.Replace(c.Request.URL.RawQuery, "meta=1", "", 1)
			relPath = "/Meta_" + username + c.Param("reqPath")
		} else {
			relPath = "/" + username + c.Param("reqPath")
		}

		remote, err := url.Parse(address + relPath)
		if utils.CheckErrorForWeb(err, "endpoints ReverseProxy2 [2]", c) {
			return
		}

		// Check file lock
		if _, lock := filer.GetFileLock(relPath); lock != "" {
			c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		//Define the director func
		//This is a good place to log, for example
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = remote.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
		}

		modifyProxyResponse(proxy, username, c)

		proxy.ServeHTTP(c.Writer, cleanProxyHeaders(c.Request))
	}
}

func DownloadFileFromFuse(c *gin.Context) {
	// Pass handling to Filer if there are some query params
	hasMeta := strings.Contains(c.Request.URL.RawQuery, "meta")
	if !hasMeta && strings.Replace(c.Request.URL.RawQuery, "meta=1", "", 1) != "" {
		c.Next()
		return
	}

	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var relPath string
	if hasMeta {
		relPath = "Meta_" + username + c.Param("reqPath")
	} else {
		relPath = username + c.Param("reqPath")
	}
	filePath := filesApi.Settings.FilerRootFolder + relPath

	// Check file lock
	if _, lock := filer.GetFileLock(relPath); lock != "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
		return
	}

	if ok, err := filesApi.Exist(filePath); err == nil && ok {
		c.File(filePath)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("No such file or another error!")})
	}

	c.AbortWithStatus(http.StatusOK)
}

func uploadFile(c *gin.Context, file *multipart.FileHeader, fileRelPath string) error {
	// Upload to temp folder and make new version if exists, and then replace the old one
	oldFilePath := filesApi.Settings.FilerRootFolder + fileRelPath

	if exist, err := filesApi.Exist(oldFilePath); err == nil && exist {
		newTempFilePath := filesApi.Settings.FilerTempFolder + fileRelPath
		filesApi.CreateDirIfNotExists(filepath.Dir(newTempFilePath))

		if err2 := c.SaveUploadedFile(file, newTempFilePath); err2 != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err2.Error()))
			return err2
		}

		if bytes.Equal(filesApi.GetFileMd5Hash(oldFilePath), filesApi.GetFileMd5Hash(newTempFilePath)) {
			_ = os.Remove(newTempFilePath)
			return nil
		}

		metaFilePath, sigPath_, currentFileVersionString, currentFileVersion, err2 := filesApi.GetFileCurrentVersion(fileRelPath)
		if err2 != nil {
			err3 := os.Remove(newTempFilePath)
			if err3 != nil {
				return err3
			}
			return err2
		}

		err = filesApi.MakeVersionDelta(newTempFilePath, oldFilePath, currentFileVersion, currentFileVersionString, metaFilePath, sigPath_)
		if err != nil {
			err3 := os.Remove(newTempFilePath)
			if err3 != nil {
				return err3
			}
			return err
		}

	repeat:
		errPath := strings.Join(strings.Split(fileRelPath, "/")[1:], "/")
		if err = filer.CheckSetCheckFileLock(fileRelPath, errPath, true); err2 == nil {

			err = exec.Command("mv", newTempFilePath, oldFilePath).Run()
			if err != nil {
				err2 = os.Remove(newTempFilePath)
				if err2 != nil {
					return err2
				}
				filer.RemoveFileLock(fileRelPath)
				return err
			}
			filer.RemoveFileLock(fileRelPath)
		} else {

			time.Sleep(1 * time.Second)
			goto repeat
		}

	} else if err == nil && !exist {
		err2 := filesApi.CreateDirIfNotExists(filepath.Dir(oldFilePath))
		if err2 != nil {
			return err2
		}

		if err2 = c.SaveUploadedFile(file, oldFilePath); err2 != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err2.Error()))
			return err2
		}
		return filesApi.GenerateFileSig(fileRelPath)
	} else {
		return err
	}

	return nil
}

func UploadFileToFuseAndMakeNewVersionIfNeeded(c *gin.Context) {
	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("upload file err: %s", err.Error())})
		return
	}

	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Check file lock
	fileRelPath := username + c.Param("reqPath")
	if _, lock := filer.GetFileLock(fileRelPath); lock != "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
		return
	}

	err = uploadFile(c, form.File["file"][0], fileRelPath)
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("upload file err")})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

func upgradeToWsAndGetUsername(c *gin.Context) (net.Conn, string) {
	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	if utils.CheckErrorForWeb(err, "api.endpoints.upgradeToWsAndGetUsername() [1]", c) {
		return nil, ""
	}

	// Get username parameter from jwt middleware
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return nil, ""
	}
	return conn, username
}

// GET (POST ws) /api/upload_files
func UploadFiles(c *gin.Context) {
	conn, username := upgradeToWsAndGetUsername(c)
	filesApi.WsReceiveFiles(conn, username)
}

// GET (POST ws) /api/upload_file
func UploadFile(c *gin.Context) {
	conn, username := upgradeToWsAndGetUsername(c)
	filesApi.WsReceiveFile(conn, username)
}

// GET (POST ws) /api/make_version_delta
func MakeVersionDelta(c *gin.Context) {
	conn, username := upgradeToWsAndGetUsername(c)
	filesApi.WsMakeVersionDelta(conn, username)
}

// GET (POST ws) /api/upload_new_file_version
func UploadNewFileVersion(c *gin.Context) {
	conn, username := upgradeToWsAndGetUsername(c)
	filesApi.WsReceiveNewFileVersion(conn, username)
}

// GET (DUAL ws) /api/download_new_file_version
func DownloadNewFileVersion(c *gin.Context) {
	conn, username := upgradeToWsAndGetUsername(c)
	filesApi.WsSendNewFileVersion(conn, username)
}

type downgradeTo struct {
	Version     int    `json:"version" binding:"required"`
	FileRelPath string `json:"rel_path" binding:"required"`
}

func DowngradeFileToVersion(c *gin.Context) {
	var json downgradeTo
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	filesApi.DowngradeFileToVersion(json.Version, username+"/"+json.FileRelPath, c)
}
