package api

import (
	"../filer"
	filesApi "../files"
	main "../main_settings"
	"../utils"
	apiCommon "./common"
	"archive/zip"
	"github.com/saracen/fastzip"
	"golang.org/x/sys/unix"
	"io"
	"mime"
	"mime/multipart"
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

type sharedFile struct {
	Path       string `json:"path" binding:"required"`       // from root folder (without first slash)
	ExpTime    string `json:"exp_time" binding:"required"`   // 0 (never) or unix time
	Type       string `json:"type" binding:"required"`       // public, group_'name'
	Permission string `json:"permission" binding:"required"` // r or rw
}

// PUT /api/shared_link
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

	// check whether file exists on Filer
	if exist, err := filesApi.Exist(filesApi.Settings.FilerRootFolder + username + "/" + json.Path); (err == nil && !exist) || err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// firstMD5 := md5.Sum([]byte(json.Path))
	hash := fmt.Sprintf("%x", md5.Sum([]byte(json.Path+utils.TokenGenerator(16))))

	var link string
	var key, value string
	var key2, linkType string

	// Check values
	if json.Permission != "r" && json.Permission != "rw" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Wrong parameters"})
		return
	}

	if json.Type != "public" && !strings.HasPrefix(json.Type, "group_") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Wrong parameters"})
		return
	}

	if json.Path[len(json.Path)-1] == '/' {
		json.Path = json.Path[:len(json.Path)-1]
	}
	//

	// Set key for CloudServerData
	if json.Type == "public" {
		key = "shl_" + hash
		linkType = ";p"
		value = username + ";" + json.Path + ";" + json.Permission + ";" + json.ExpTime + linkType // last is a link type
		link = hash + "a"                                                                          // public link
	} else if strings.HasPrefix(json.Type, "group") {
		key = "shl_" + hash
		linkType = ";g_" + json.Type[6:]
		value = username + ";" + json.Path + ";" + json.Permission + ";" + json.ExpTime + linkType
		link = hash + "b" // group link
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong link type"})
	}

	// Set key for username
	key2 = "shl_" + json.Path

	// CloudServerData
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

	// username
	if has, err := apiCommon.UserStates.HasKey(username, key2); !has {
		err2 := apiCommon.UserStates.SetKey(username, key2, link+linkType)
		if utils.CheckErrorForWeb(err2, "endpoints CreateSharedLinks [3]", c) {
			return
		}
	} else {
		if utils.CheckErrorForWeb(err, "endpoints CreateSharedLinks [4]", c) {
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"link": link, "type": linkType})
}

// GET /api/shared_link/*reqPath
func GetSharedLink(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	linkWithType, err := apiCommon.UserStates.GetKey(username, "shl_"+c.Param("reqPath")[1:])
	if err != nil && err.Error() == "redigo: nil returned" {
		c.Status(http.StatusNotFound)
		return
	} else if utils.CheckErrorForWeb(err, "api endpoints GetSharedLink [1]", c) {
		return
	}

	split := strings.Split(linkWithType, ";")
	c.JSON(http.StatusOK, gin.H{"link": split[0], "type": split[1]})
}

type sFile2 struct {
	Path     string `json:"path"`
	LinkHash string `json:"link"`
}

// DELETE /api/shared_link
func RemoveSharedLink(c *gin.Context) {
	var json sFile2
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var hash, path string
	var err error
	if len(json.Path) > 0 && len(json.LinkHash) > 16 {
		path = json.Path
		hash = json.LinkHash[:len(json.LinkHash)-1]

	} else if len(json.Path) > 0 {
		path = json.Path
		hash, err = apiCommon.UserStates.GetKey(username, path)
		if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [1]", c) {
			return
		}

	} else if len(json.LinkHash) > 16 {
		hash = json.LinkHash[:len(json.LinkHash)-1]
		path, err = apiCommon.UserStates.GetKey("CloudServerData", hash)
		if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [2]", c) {
			return
		}
		path = strings.Split(path, ";")[1]

	} else {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	err = apiCommon.UserStates.DelKey("CloudServerData", "shl_"+hash)
	if utils.CheckErrorForWeb(err, "api endpoints RemoveSharedLink [3]", c) {
		return
	}

	err = apiCommon.UserStates.DelKey(username, "shl_"+path)
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
	//delete(req.Header, "X-Auth-Token")
	delete(req.Header, "X-Csrf-Token")
	//delete(req.Header, "X-Refresh-Token")
	delete(req.Header, "Set-Cookie")

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

func ModifyProxyRequest(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	noTagging := !strings.Contains(c.Request.URL.RawQuery, "tagging")

	// DELETE method
	if c.Request.Method == "DELETE" && noTagging {
		relPath := username + c.Param("reqPath")

		// Check file lock
		if _, lock := filer.GetFileLock(relPath); lock != "" {
			c.AbortWithStatusJSON(http.StatusLocked, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
			return
		}

		metaPath := filesApi.Settings.FilerRootFolder + "Meta_" + relPath
		regularPath := filesApi.Settings.FilerRootFolder + relPath

		if info, err := os.Stat(filesApi.Settings.FilerRootFolder + relPath); err == nil && info.IsDir() {
			// Remove meta
			err2 := filesApi.RemoveContents(metaPath)
			if err2 == nil && filepath.Base(metaPath) != ("Meta_"+username) {
				err = os.Remove(metaPath)
				if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [1]", c) {
					return
				}
			} else if err2.(*os.PathError).Err != unix.ENOENT {
				if utils.CheckErrorForWeb(err2, "endpoints ModifyProxyRequest [2]", c) {
					return
				}
			}

			// Remove regular
			err2 = filesApi.RemoveContents(regularPath)
			if err2 == nil && filepath.Base(regularPath) != username {
				err = os.Remove(regularPath)
				if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [1]", c) {
					return
				}
			} else if err2.(*os.PathError).Err != unix.ENOENT {
				if utils.CheckErrorForWeb(err2, "endpoints ModifyProxyRequest [2]", c) {
					return
				}
			}

			c.AbortWithStatus(http.StatusOK)
			return

		} else if err == nil && !info.IsDir() && noTagging {
			// Remove meta file
			err = filesApi.RemoveFileMetadata(relPath)
			if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [3]", c) {
				return
			}

			// Remove file
			err = os.Remove(regularPath)
			if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [3]", c) {
				return
			}

			c.AbortWithStatus(http.StatusOK)
			return

		} else if info == nil {
			c.AbortWithStatus(http.StatusOK)
			return
		} else if err.(*os.PathError).Err != unix.ENOENT {
			if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [4]", c) {
				return
			}
		}

		// POST method
	} else if c.Request.Method == "PUT" && noTagging {
		// Create new dir
		err := filesApi.CreateDirIfNotExists(filesApi.Settings.FilerRootFolder + username + c.Param("reqPath"))
		if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [5]", c) {
			return
		} // err != nil && err.(*os.PathError).Err != unix.EEXIST &&

		// Create new meta dir
		err = filesApi.CreateDirIfNotExists(filesApi.Settings.FilerRootFolder + "Meta_" + username + c.Param("reqPath"))
		if utils.CheckErrorForWeb(err, "endpoints ModifyProxyRequest [6]", c) {
			return
		}

		c.AbortWithStatusJSON(http.StatusCreated, gin.H{})
		return
	}

	c.Next()
}

func modifyProxyResponse(proxy *httputil.ReverseProxy, username string, c *gin.Context) {
	if c.Request.Method == "POST" {
		proxy.ModifyResponse = func(resp *http.Response) error {
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				return filesApi.GenerateFileSig(username + c.Param("reqPath"))
			}
			return nil
		}
	} // else if c.Request.Method == "GET" {
	//	proxy.ModifyResponse = func(resp *http.Response) error {
	//		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
	//			resp.Header.Set("Content-Type", "application/octet-stream")
	//			return nil
	//		}
	//		return nil
	//	}
	//}
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
			relPath = "Meta_" + username + c.Param("reqPath")
		} else {
			relPath = username + c.Param("reqPath")
		}

		remote, err := url.Parse(address + relPath)
		if utils.CheckErrorForWeb(err, "endpoints ReverseProxy2 [2]", c) {
			return
		}

		// Check file lock (if it's not tagging)
		if !strings.Contains(c.Request.URL.RawQuery, "tagging") {
			if _, lock := filer.GetFileLock(relPath); lock != "" {
				c.AbortWithStatusJSON(http.StatusLocked, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
				return
			}
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
		c.Abort()
	}
}

// GET /api/filer/*reqPath
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
		c.AbortWithStatusJSON(http.StatusLocked, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
		return
	}

	if ok, err := filesApi.Exist(filePath); err == nil && ok {
		c.File(filePath)
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("No such file or another error!")})
	}

	c.AbortWithStatus(http.StatusOK)
}

func uploadFile(c *gin.Context, relPath string) error {
	// Upload to temp folder (it's already in multipart) and make new version if exists, and then replace the old one
	oldFilePath := filesApi.Settings.FilerRootFolder + relPath

	if exist, err := filesApi.Exist(oldFilePath); err == nil && exist {
		// Temp file
		newTempFilePath := filesApi.Settings.FilerTempFolder + relPath
		err = filesApi.CreateDirIfNotExists(filepath.Dir(newTempFilePath))
		if err != nil {
			return err
		}

		tempFile, err2 := os.OpenFile(newTempFilePath, os.O_CREATE|os.O_WRONLY, 0600)
		if err2 != nil {
			return err2
		}
		//

		// Upload to temp file
		_, params, err3 := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
		if err3 != nil {
			return err3
		}
		mr := multipart.NewReader(c.Request.Body, params["boundary"])
		for {
			part, err4 := mr.NextPart()
			if err4 == io.EOF || part == nil {
				break
			}

			_, err5 := io.Copy(tempFile, part)
			if err5 != nil {
				if err5 == io.ErrUnexpectedEOF {
					_ = tempFile.Close()
					_ = os.Remove(newTempFilePath)
				} else {
					return err5
				}
			}
		}
		err = tempFile.Close()
		if err != nil && err.(*os.PathError).Err.Error() == "file already closed" {
			return nil
		} else if err != nil && err.(*os.PathError).Err.Error() != "file already closed" {
			return err
		}
		//

		if bytes.Equal(filesApi.GetFileMd5Hash(oldFilePath), filesApi.GetFileMd5Hash(newTempFilePath)) {
			_ = os.Remove(newTempFilePath)
			return nil
		}

		errPath := strings.Join(strings.Split(relPath, "/")[1:], "/")
	repeat:
		if err = filer.CheckSetCheckFileLock(relPath, errPath, true); err == nil {

			metaFilePath, sigPath_, currentFileVersionString, currentFileVersion, err5 := filesApi.GetFileCurrentVersion(relPath)
			if err5 != nil {
				_ = os.Remove(newTempFilePath)
				//if err4 != nil {
				//	return err4
				//}
				return err5
			}

			err = filesApi.MakeVersionDelta(newTempFilePath, oldFilePath, currentFileVersion, currentFileVersionString, metaFilePath, sigPath_)
			if err != nil {
				_ = os.Remove(newTempFilePath)
				//if err3 != nil {
				//	return err3
				//}
				return err
			}

			err = exec.Command("mv", newTempFilePath, oldFilePath).Run()
			if err != nil {
				_ = os.Remove(newTempFilePath)
				//if err2 != nil {
				//	return err2
				//}
				filer.RemoveFileLock(relPath)
				return err
			}

			filer.RemoveFileLock(relPath)

		} else {
			time.Sleep(1 * time.Second)
			goto repeat
		}

	} else if err == nil && !exist {
		c.Next() // pass it to Filer
	} else {
		return err
	}

	return nil
}

// POST /api/filer/*reqPath
func UploadFileToFuseAndMakeNewVersionIfNeeded(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Check file lock
	fileRelPath := username + c.Param("reqPath")
	if _, lock := filer.GetFileLock(fileRelPath); lock != "" {
		c.AbortWithStatusJSON(http.StatusLocked, gin.H{"status": fmt.Sprintf("File is locked. Try later.")})
		return
	}

	err := uploadFile(c, fileRelPath)
	if err != nil {
		fmt.Println(err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"status": fmt.Sprintf("upload file err")})
		return
	}

	c.AbortWithStatus(http.StatusCreated)
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

// GET /api/version/*reqPath
func ListFileVersions(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	relPath := username + c.Param("reqPath")
	if info, err := os.Stat(filesApi.Settings.FilerRootFolder + relPath); err == nil && info.IsDir() {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	} else if utils.CheckErrorForWeb(err, "api.ListFileVersions [1]", c) {
		return
	}

	metaFilePath, _, _, currentFileVersion, err := filesApi.GetFileCurrentVersion(relPath)
	if utils.CheckErrorForWeb(err, "api.ListFileVersions [2]", c) {
		return
	}

	if currentFileVersion == 1 {
		c.JSON(http.StatusOK, gin.H{"versions": []string{}})
	} else {
		versions_, _ := filepath.Glob(metaFilePath + ".delta.v*")
		versions := []string{"1;_"}

		for i := 1; i < len(versions_); i++ {
			fileInfo, err2 := os.Stat(versions_[i-1])
			if utils.CheckErrorForWeb(err2, "api.ListFileVersions [3]", c) {
				return
			}

			split := strings.Split(versions_[i], ".") // for getting version number
			versions = append(versions, fmt.Sprintf("%s;%s",
				split[len(split)-1][1:], fileInfo.ModTime().Format("02.01.2006, 15:04")))
		}
		versions = append(versions, fmt.Sprintf("%d;%s", len(versions)+1, "текущая"))

		c.JSON(http.StatusOK, gin.H{"versions": versions})
	}
}

type downgradeTo struct {
	Version     int    `json:"version" binding:"required"`
	FileRelPath string `json:"rel_path" binding:"required"`
}

// PATCH /api/version
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

// GET /api/zip/filer/*reqPath
func CreateZipFromFolder(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	reqPath := c.Param("reqPath")
	dirPath := filesApi.Settings.FilerRootFolder + username + reqPath
	if info, err := os.Stat(dirPath); err == nil && !info.IsDir() {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	} else if utils.CheckErrorForWeb(err, "api endpoints CreateZipFromFolder [1]", c) {
		return
	}

	dirName := filepath.Base(reqPath)
	if dirName == "" {
		dirName = "folder.zip"
	} else {
		dirName += ".zip"
	}
	c.Writer.Header().Set("Content-type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", dirName))

	// Create new Archiver
	a, err := fastzip.NewArchiver(c.Writer, dirPath, fastzip.WithArchiverConcurrency(2))
	if utils.CheckErrorForWeb(err, "api endpoints CreateZipFromFolder [2]", c) {
		return
	}
	defer a.Close()

	// Register a non-default level compressor
	a.RegisterCompressor(zip.Deflate, fastzip.FlateCompressor(3))

	// Walk directory, adding the files we want to add
	files := make(map[string]os.FileInfo)
	err = filepath.Walk(dirPath, func(pathname string, info os.FileInfo, err error) error {
		files[pathname] = info
		return nil
	})

	// Archive
	if err = a.Archive(c, files); err != nil {
		panic(err)
	}

	c.Status(http.StatusOK)
}
