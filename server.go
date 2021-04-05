package main

import (
	endpoints "./src/api"
	apiCommon "./src/api/common"
	"net/http"

	"fmt"

	"github.com/gin-gonic/gin"
)

//var (
//	filesAddr = flag.String("listen", ":50000", "port to listen to")
//)

//var CookieStore = cookie.NewStore([]byte("gVMJKf@,t{ER4xgf=*'%#n#8Hk'+'9(2"),
//								  []byte("Axq4Rqh,CFru8K]DaJ)&tR{tww,Jc9Q9"))

func main() {
	fmt.Println("Server started...")

	// go fileHandlers.ReceiveFile(*filesAddr, "./test_data/koko.rar")
	//sf := handler.NewSendFile(0.05)
	//sf.SendFile("./test_data/mek.delta", "192.168.0.244", 60000)
	//handler.GetFileHash("./test_data/mek.delta")

	router := gin.New()
	// Logging middleware
	router.Use(gin.Logger())
	router.Use(apiCommon.SecureMiddleware())

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	//router.GET("/oauth/google/login", oauth.GoogleLoginHandler)
	//router.GET("/oauth/google/auth", oauth.GoogleAuthHandler)

	publicApi := router.Group("/api")
	{
		publicApi.POST("/register", endpoints.Register)
		publicApi.POST("/confirm_username", endpoints.ConfirmUser)
		publicApi.POST("/login", endpoints.Login)
	}

	api := router.Group("/api")
	api.Use(apiCommon.JwtMiddleware(), apiCommon.PermissionMiddleware())
	{
		api.GET("/restricted_hello", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"hello,": "world!"})
		})

		api.GET("/upload_files", endpoints.UploadFilesWs)
	}

	// Recovery middleware
	router.Use(gin.Recovery())

	_ = router.Run(":8080")
	// log.Fatal(autotls.Run(router, "mgtu-diploma.tk", "5bdb593d765bb4.localhost.run"))
}
