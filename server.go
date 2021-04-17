package main

import (
	apiEndpoints "./src/api"
	apiCommon "./src/api/common"
	"net/http"

	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Server started...")

	router := gin.New()
	// Logging middleware
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

	filer := "http://13.53.193.254:8888"

	// Public router
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, public world!")
	})
	router.GET("/public_share/:link", apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(filer, "Share"))

	// Private router
	privateRouter := router.Group("/")
	privateRouter.Use(apiCommon.JwtMiddleware(), apiCommon.PermissionMiddleware())
	{
		privateRouter.GET("/secure_share/:link", apiEndpoints.GetPathFromLink, apiEndpoints.ReverseProxy(filer, "Share"))
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

		api.GET("/sync_files", apiEndpoints.SyncFilesWs)

		api.GET("/filer/*proxyPath", apiEndpoints.ReverseProxy(filer, "Filer"))
		api.POST("/filer/*proxyPath", apiEndpoints.ReverseProxy(filer, "Filer"))

		api.GET("/shared_link", apiEndpoints.CreateSharedLink)
		api.DELETE("/shared_link", apiEndpoints.RemoveSharedLink)
	}

	// Recovery middleware
	router.Use(gin.Recovery())

	_ = router.Run(":8080")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk"))
}
