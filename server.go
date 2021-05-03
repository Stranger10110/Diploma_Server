package main

import (
	apiEndpoints "./src/api"
	apiCommon "./src/api/common"
	s "./src/main_settings"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Server started...")

	router := gin.New()
	// Logging middleware
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

	// TODO: make config file
	// filerAddr := "http://13.53.193.254:8888"

	// Public router
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, public world!")
	})
	router.GET("/public_share/:link",
		apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(s.Settings.FilerAddress, false))

	router.GET("/seaweedfsstatic/*reqPath", apiEndpoints.ReverseProxy2(s.Settings.FilerAddress, false, true, false))
	filer := router.Group("/filer") // TODO: move into private
	{
		filer.GET("/*reqPath", apiEndpoints.ReverseProxy2(s.Settings.FilerAddress, false, true, false))
		filer.POST("/*reqPath", apiEndpoints.ReverseProxy2(s.Settings.FilerAddress, false, true, false))
	}

	// Private router
	privateRouter := router.Group("/")
	privateRouter.Use(apiCommon.JwtMiddleware(), apiCommon.PermissionMiddleware())
	{
		privateRouter.GET("/secure_share/:link",
			apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(s.Settings.FilerAddress, false))
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

		api.GET("/filer", apiEndpoints.GetFilerInfoFromHeader,
			apiEndpoints.ReverseProxy("http://"+s.Settings.FilerAddress, false))
		api.POST("/filer", apiEndpoints.GetFilerInfoFromHeader,
			apiEndpoints.ReverseProxy("http://"+s.Settings.FilerAddress, true))
		api.PUT("/filer", apiEndpoints.GetFilerInfoFromHeader,
			apiEndpoints.ReverseProxy("http://"+s.Settings.FilerAddress, false))
		api.DELETE("/filer", apiEndpoints.GetFilerInfoFromHeader,
			apiEndpoints.ReverseProxy("http://"+s.Settings.FilerAddress, false))
		api.HEAD("/filer", apiEndpoints.GetFilerInfoFromHeader,
			apiEndpoints.ReverseProxy("http://"+s.Settings.FilerAddress, false))
	}

	// Recovery middleware
	router.Use(gin.Recovery())

	_ = router.Run(":8080")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk"))
}
