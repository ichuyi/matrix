package handler

import (
	"github.com/gin-gonic/gin"
	_ "matrix/dao"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	v1 := r.Group("/matrix")
	{
		v1.POST("/add", addMatrix)
		v1.GET("/get", getMatrix)
	}
	return r
}
