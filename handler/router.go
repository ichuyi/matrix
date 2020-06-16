package handler

import (
	"github.com/gin-gonic/gin"
	_ "matrix/dao"
)

func InitRouter() *gin.Engine {
	//gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	v1 := r.Group("/matrix")
	{
		v1.POST("/resource", resource)
		v1.POST("/forward", forward)
		v2 := v1.Group("/operation")
		{
			v2.POST("/all", operateAll)
			v2.POST("/setSpeed", setSpeed)
			v2.POST("/motor", motorAction)
			v2.POST("/setWIFI", setWIFI)
			v2.POST("/setTCP", setTCP)
		}
	}
	return r
}
