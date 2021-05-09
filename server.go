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
// TODO: minify js and css
// TODO: "remember me" option

// Now
// TODO: make download of a file
// TODO: make new version on upload if needed
// TODO: when deleting files, delete all meta

func main() {
	fmt.Println("Server started...")

	method := "http://"

	router := gin.New()
	router.LoadHTMLGlob("./src/html/templates/*/*")
	// router.LoadHTMLFiles("./src/html/login/login.")

	//router.Use(gzip.Gzip(gzip.DefaultCompression)) // TODO: test if needed
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

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
	//

	// Private router
	privateRouter := router.Group("/secure")
	privateRouter.Use(apiCommon.JwtMiddleware(), apiCommon.PermissionMiddleware())
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
	api.Use(apiCommon.JwtMiddleware(), apiCommon.PermissionMiddleware())
	{
		api.GET("/restricted_hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"hello,": "world!"})
		})

		api.GET("/upload_files", apiEndpoints.UploadFiles)                         // POST
		api.GET("/upload_file", apiEndpoints.UploadFile)                           // POST
		api.GET("/make_version_delta", apiEndpoints.UploadSignatureForNewVersion)  // POST
		api.GET("/upload_new_file_version", apiEndpoints.UploadNewFileVersion)     // POST
		api.GET("/download_new_file_version", apiEndpoints.DownloadNewFileVersion) // POST

		api.POST("/downgrade_to", apiEndpoints.DowngradeFileToVersion)

		api.GET("/shared_link", apiEndpoints.CreateSharedLink)
		api.DELETE("/shared_link", apiEndpoints.RemoveSharedLink)

		api.GET("/filer/*reqPath", apiEndpoints.ReverseProxy2(method+s.Settings.FilerAddress, false, false, true))
		api.POST("/filer/*reqPath", apiEndpoints.ReverseProxy2(method+s.Settings.FilerAddress, true, false, true))
		api.PUT("/filer/*reqPath", apiEndpoints.ReverseProxy2(method+s.Settings.FilerAddress, false, false, true))
		api.DELETE("/filer/*reqPath", apiEndpoints.ReverseProxy2(method+s.Settings.FilerAddress, false, false, true))
		api.HEAD("/filer/*reqPath", apiEndpoints.ReverseProxy2(method+s.Settings.FilerAddress, false, false, true))
	}

	router.Use(gin.Recovery())

	_ = router.Run(":8080")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk"))
}
