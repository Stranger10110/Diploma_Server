package main

import (
	apiEndpoints "./src/api"
	apiCommon "./src/api/common"
	"./src/html"
	s "./src/main_settings"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// Later
// TODO: logout button
// TODO: minify js and css
// TODO: "remember me" option
// TODO: list folders first (maybe)
// TODO: "Пусто" text when no files
// DONE: icons for file types
// TODO: setting to disable popups

// TODO: fix prometheus or disable it

// Now
// DONE: when deleting files, delete all meta
// DONE: make new version on html upload if needed
// DONE: delete button
// DONE~: "new folder" button
// TODO: fix shared links
// DONE: check file lock before uploading
// DONE: check file lock before downloading

// DONE~: add confirmation for creating signature
// TODO: set whitelists

// 13.53.193.254
// "13.53.193.254:9333,13.53.193.254:9334,13.53.193.254:9335"

func main() {
	fmt.Println("Server started...")

	router := gin.New()
	router.LoadHTMLGlob("./src/html/templates/*/*")
	// router.LoadHTMLFiles("./src/html/login/login.")

	//router.Use(gzip.Gzip(gzip.DefaultCompression)) // TODO: test if needed
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = 5242880 // 5MB

	// Public router
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", gin.H{})
	})

	router.GET("/share/:link",
		apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(s.Settings.FilerAddress, false))

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})
	router.GET("/filer/*path", func(c *gin.Context) {
		c.HTML(http.StatusOK, "filer.html", gin.H{})
	})

	router.StaticFile("/login.css", "./src/html/templates/login/login.css")
	router.StaticFile("/login.js", "./src/html/templates/login/login.js")
	router.StaticFile("/filer.css", "./src/html/templates/filer/filer.css")
	router.StaticFile("/filer.js", "./src/html/templates/filer/filer.js")
	router.StaticFile("/jquery_binarytransport.js", "./src/html/templates/filer/jquery_binarytransport.js")
	//

	// Private router
	privateRouter := router.Group("/secure")
	privateRouter.Use(apiCommon.JwtMiddleware()) //, apiCommon.PermissionMiddleware())
	{
		privateRouter.GET("/share/:link",
			apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(s.Settings.FilerAddress, false))

		privateRouter.GET("/filer/*reqPath", html.FilerListing)
	}

	// Public API
	publicApi := router.Group("/api")
	{
		publicApi.POST("/register", apiEndpoints.Register)
		publicApi.POST("/confirm_username", apiEndpoints.ConfirmUser)
		publicApi.POST("/login", apiEndpoints.Login)
	}

	// Private API
	api := router.Group("/api")
	api.Use(apiCommon.JwtMiddleware()) //, apiCommon.PermissionMiddleware())
	{
		api.GET("/restricted_hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"hello,": "world!"})
		})

		api.GET("/upload_files", apiEndpoints.UploadFiles)                         // POST
		api.GET("/upload_file", apiEndpoints.UploadFile)                           // POST
		api.GET("/make_version_delta", apiEndpoints.MakeVersionDelta)              // POST
		api.GET("/upload_new_file_version", apiEndpoints.UploadNewFileVersion)     // POST
		api.GET("/download_new_file_version", apiEndpoints.DownloadNewFileVersion) // POST

		api.POST("/downgrade_to", apiEndpoints.DowngradeFileToVersion)

		api.GET("/shared_link", apiEndpoints.CreateSharedLink)
		api.DELETE("/shared_link", apiEndpoints.RemoveSharedLink)

		filer := api.Group("/filer")
		{
			filer.GET("/*reqPath", apiEndpoints.DownloadFileFromFuse, apiEndpoints.ReverseProxy2(s.Settings.Method+s.Settings.FilerAddress))
			filer.POST("/*reqPath", apiEndpoints.UploadFileToFuseAndMakeNewVersionIfNeeded, apiEndpoints.ReverseProxy2(s.Settings.Method+s.Settings.FilerAddress))
			filer.PUT("/*reqPath", apiEndpoints.ModifyProxyRequest, apiEndpoints.ReverseProxy2(s.Settings.Method+s.Settings.FilerAddress))
			filer.DELETE("/*reqPath", apiEndpoints.ModifyProxyRequest, apiEndpoints.ReverseProxy2(s.Settings.Method+s.Settings.FilerAddress))
			filer.HEAD("/*reqPath", apiEndpoints.ReverseProxy2(s.Settings.Method+s.Settings.FilerAddress))
		}
	}

	router.Use(gin.Recovery())

	_ = router.Run(":80")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk"))
}
