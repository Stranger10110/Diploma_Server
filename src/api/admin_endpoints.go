package api

import (
	apiCommon "./common"
	"github.com/gin-gonic/gin"
	// "net/http"
	"../utils"
)

//type username struct {
//	Username string `json:"username" binding:"required"`
//}
//
//// /api/username_exists
//func UsernameExists(c *gin.Context) {
//	var json username
//	if err := c.ShouldBindJSON(&json); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//
//	if userExists(json.Username) {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "username already exists"})
//		return
//	}
//}

func SetGroupForUser(username string, group string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// There won't be so many groups, so we're saving time instead of RAM (instead of "groups": "g1;g2;...")
		err := apiCommon.UserStates.SetKey(username, "grp_"+group, "")
		utils.CheckErrorForWeb(err, "admin endpoints SetGroupForUser [1]", c)
	}
}
