package api

import (
	apiCommon "./common"
	"github.com/gin-gonic/gin"
	"net/http"

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

func CheckAdminRights(c *gin.Context) {
	username := GetUserName(c)
	if username == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	if !apiCommon.UserStates.AdminRights(username) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.Next()
}

/*
	Groups: testing
*/
type userGroup struct {
	Username string `json:"username" binding:"required"`
	Group    string `json:"group" binding:"required"`
}

// PUT /api/admin/users/group
func SetGroupForUser(c *gin.Context) {
	var json userGroup
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := apiCommon.UserStates.SetKey(json.Username, "group_"+json.Group, "")
	utils.CheckErrorForWeb(err, "admin endpoints SetGroupForUser [1]", c)
}

// DELETE /api/admin/users/group
func RemoveGroupFromUser(c *gin.Context) {
	var json userGroup
	if err := c.ShouldBindJSON(&json); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := apiCommon.UserStates.DelKey(json.Username, "group_"+json.Group)
	utils.CheckErrorForWeb(err, "admin endpoints RemoveGroupFromUser [1]", c)
}
