package handler

import (
	"github.com/gin-gonic/gin"
	_ "matrix/dao"
)

func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	v1 := r.Group("/matrix")
	{
		v1.POST("/resource", resource)
	}
	return r
}
