package handler

import (
	"github.com/gin-gonic/gin"
	_ "matrix/dao"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	//gin.SetMode(gin.ReleaseMode)
	v1 := r.Group("/matrix")
	{
		v1.POST("/resource", resource)
	}
	return r
}
