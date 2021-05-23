package main

import (
	apiEndpoints "./src/api"
	apiCommon "./src/api/common"
	"./src/html"
	s "./src/main_settings"
	"net"

	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Later
// TODO: logout button
// TODO: minify js and css
// TODO: "remember me" option
// DONE: list folders first
// TODO: "Пусто" text when no files
// DONE: icons for file types
// TODO: setting to disable popups

// TODO: fix prometheus or disable it

// Now
// DONE: when deleting files, delete all meta
// DONE: make new version on html upload if needed
// DONE: delete button
// DONE: "new folder" button
// DONE: fix shared links
// DONE: check file lock before uploading
// DONE: check file lock before downloading
// DONE~: add confirmation for creating signature

// TODO: add check for write permissions for shared files
// TODO: download zip folder

// TODO: copy/move functionality
// TODO: set whitelists

// 13.53.193.254
// "13.53.193.254:9333,13.53.193.254:9334,13.53.193.254:9335"

func main() {
	fmt.Println("Server started...")

	router := gin.New()
	router.LoadHTMLGlob("./src/html/templates/*/*.html")
	// router.LoadHTMLFiles("./src/html/login/login.")

	// router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithDecompressFn(gzip.DefaultDecompressHandle))) // TODO: test if needed
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

	// Set a lower memory limit for multipart forms (default is 32 MiB)
	router.MaxMultipartMemory = 5242880 // 5MB

	// Public router
	router.GET("/", func(c *gin.Context) { c.HTML(http.StatusOK, "home.html", gin.H{}) })
	router.GET("/login", func(c *gin.Context) { c.HTML(http.StatusOK, "login.html", gin.H{}) })
	router.GET("/filer/*path", func(c *gin.Context) { c.HTML(http.StatusOK, "filer.html", gin.H{}) })
	router.GET("/share/:link", func(c *gin.Context) { c.HTML(http.StatusOK, "share.html", gin.H{}) })
	router.GET("/shared/content/:link/*reqPath", apiEndpoints.SetInfoFromLink, html.FilerListing)
	router.NoRoute(func(c *gin.Context) { c.HTML(http.StatusOK, "not_found.html", gin.H{}) })

	src := router.Group("/src")
	{
		src.StaticFile("/login.css", "./src/html/templates/login/login.css")
		src.StaticFile("/login.js", "./src/html/templates/login/login.js")
		src.StaticFile("/filer.css", "./src/html/templates/filer/filer.css")
		src.StaticFile("/filer.js", "./src/html/templates/filer/filer.js")
		src.StaticFile("/jquery_binarytransport.js", "./src/html/templates/filer/jquery_binarytransport.js")
		src.StaticFile("/share.js", "./src/html/templates/share/share.js")
		src.StaticFile("/share.css", "./src/html/templates/share/share.css")
		src.StaticFile("/not_found.js", "./src/html/templates/not_found/not_found.js")
		src.StaticFile("/not_found.css", "./src/html/templates/not_found/not_found.css")

		img := src.Group("/img")
		{
			img.StaticFile("/404.svg", "./src/html/templates/not_found/img/404.svg")
			img.StaticFile("/rocket.svg", "./src/html/templates/not_found/img/rocket.svg")
			img.StaticFile("/earth.svg", "./src/html/templates/not_found/img/earth.svg")
			img.StaticFile("/moon.svg", "./src/html/templates/not_found/img/moon.svg")
			img.StaticFile("/astronaut.svg", "./src/html/templates/not_found/img/astronaut.svg")
		}
	}
	//

	// Private router
	privateRouter := router.Group("/secure")
	privateRouter.Use(apiCommon.JwtMiddleware()) //, apiCommon.PermissionMiddleware())
	{
		privateRouter.GET("/shared/content/:link/*reqPath", apiEndpoints.SetInfoFromLink, html.FilerListing)
		privateRouter.GET("/filer/*reqPath", html.FilerListing)
	}

	reverseProxy := apiEndpoints.ReverseProxy2(s.Settings.Method + s.Settings.FilerAddress)

	// Public API
	publicApi := router.Group("/api/public")
	{
		publicApi.POST("/register", apiEndpoints.Register)
		publicApi.POST("/confirm_username", apiEndpoints.ConfirmUser)
		publicApi.POST("/login", apiEndpoints.Login)

		publicApi.GET("/zip/shared/filer/:link/*reqPath", apiEndpoints.SetInfoFromLink, apiEndpoints.CreateZipFromFolder)
		share := publicApi.Group("/shared/filer/:link/*reqPath")
		share.Use(apiEndpoints.SetInfoFromLink)
		{
			share.GET("", apiEndpoints.DownloadFileFromFuse, reverseProxy)
			share.POST("", apiEndpoints.UploadFileToFuseAndMakeNewVersionIfNeeded, reverseProxy)
			share.PUT("", apiEndpoints.ModifyProxyRequest, reverseProxy)
		}
	}

	// Private API
	api := router.Group("/api")
	api.Use(apiCommon.JwtMiddleware()) //, apiCommon.PermissionMiddleware())
	{
		admin := api.Group("/admin")
		admin.Use(apiEndpoints.CheckAdminRights)
		{
			admin.PUT("/users/group", apiEndpoints.SetGroupForUser)
			admin.DELETE("/users/group", apiEndpoints.RemoveGroupFromUser)
		}

		api.GET("/upload_files", apiEndpoints.UploadFiles)                         // POST
		api.GET("/upload_file", apiEndpoints.UploadFile)                           // POST
		api.GET("/make_version_delta", apiEndpoints.MakeVersionDelta)              // POST
		api.GET("/upload_new_file_version", apiEndpoints.UploadNewFileVersion)     // POST
		api.GET("/download_new_file_version", apiEndpoints.DownloadNewFileVersion) // POST

		api.POST("/downgrade_to", apiEndpoints.DowngradeFileToVersion)

		api.GET("/shared_link/*reqPath", apiEndpoints.GetSharedLink)
		api.PUT("/shared_link", apiEndpoints.CreateSharedLink)
		api.DELETE("/shared_link", apiEndpoints.RemoveSharedLink)

		api.GET("/zip/filer/*reqPath", apiEndpoints.CreateZipFromFolder)
		filer := api.Group("/filer/*reqPath")
		{
			filer.GET("", apiEndpoints.DownloadFileFromFuse, reverseProxy)
			filer.POST("", apiEndpoints.UploadFileToFuseAndMakeNewVersionIfNeeded, reverseProxy)
			filer.PUT("", apiEndpoints.ModifyProxyRequest, reverseProxy)
			filer.DELETE("", apiEndpoints.ModifyProxyRequest, reverseProxy)
			filer.HEAD("", reverseProxy)
		}

		api.GET("/zip/shared/filer/:link/*reqPath", apiEndpoints.SetInfoFromLink, apiEndpoints.CreateZipFromFolder)
		share := api.Group("/shared/filer/:link/*reqPath")
		share.Use(apiEndpoints.SetInfoFromLink)
		{
			share.GET("", apiEndpoints.DownloadFileFromFuse, reverseProxy)
			share.POST("", apiEndpoints.UploadFileToFuseAndMakeNewVersionIfNeeded, reverseProxy)
			share.PUT("", apiEndpoints.ModifyProxyRequest, reverseProxy)
		}
	}

	router.Use(gin.Recovery())

	ctx, cancel := context.WithCancel(context.Background())
	httpServer := &http.Server{
		Addr:        ":80",
		Handler:     router,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}
	// Run server
	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			// it is fine to use Fatal here because it is not main goroutine
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGTERM,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	<-signalChan
	log.Print("os.Interrupt - shutting down...\n")

	go func() {
		<-signalChan
		log.Fatal("os.Kill - terminating...\n")
	}()

	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(gracefulCtx); err != nil {
		log.Printf("shutdown error: %v\n", err)
		defer os.Exit(1)
		return
	} else {
		log.Printf("gracefully stopped\n")
	}

	// manually cancel context if not using httpServer.RegisterOnShutdown(cancel)
	cancel()

	defer os.Exit(0)
	return

	// _ = router.Run(":80")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk"))
}
