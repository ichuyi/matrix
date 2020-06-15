package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"matrix/dao"
	"matrix/model"
	"matrix/socket"
	"matrix/util"
	"net/http"
	"sync"
)

var current int64 = 0
var lock = new(sync.Mutex)
var unityServer *socket.TcpServer
var forwardServer *socket.TcpServer

type MatrixReq struct {
	Res string `json:"res"`
}

func init() {
	unityServer = socket.NewServer(util.ConfigInfo.Socket.UnityLaddr, util.ConfigInfo.Socket.Duration, func(s string) (bool, string) {
		return true, s
	})
	forwardServer = socket.NewServer(util.ConfigInfo.Socket.ForwardLaddr, 0, func(s string) (bool, string) {
		return true, s
	})
}
func resource(ctx *gin.Context) {
	r := MatrixReq{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	if r.Res == "forward" {
		m, err := dao.GetMatrix(current, 0)
		if err != nil {
			ctx.JSON(http.StatusOK, util.FailedResponse(GetMatrix, GetMatrixMsg))
			return
		}
		lock.Lock()
		current = m.Id
		lock.Unlock()
		ctx.JSON(http.StatusOK, util.OKResponse(m.MatrixInfo))
		go func() {
			_, s := util.CheckArr(m.MatrixInfo)
			unityServer.Message <- "matrix " + s
		}()
	} else if r.Res == "backend" {
		m, err := dao.GetMatrix(current, 1)
		if err != nil {
			ctx.JSON(http.StatusOK, util.FailedResponse(GetMatrix, GetMatrixMsg))
			return
		}
		lock.Lock()
		current = m.Id
		lock.Unlock()
		ctx.JSON(http.StatusOK, util.OKResponse(m.MatrixInfo))
		go func() {
			_, s := util.CheckArr(m.MatrixInfo)
			unityServer.Message <- "matrix " + s
		}()
	} else if ok, s := util.CheckArr(r.Res); ok {
		m := &model.Matrix{
			MatrixInfo: r.Res,
		}
		err := dao.AddMatrix(m)
		if err != nil {
			ctx.JSON(http.StatusOK, util.FailedResponse(AddMatrix, AddMatrixMsg))
			return
		}
		lock.Lock()
		current = m.Id
		lock.Unlock()
		ctx.JSON(http.StatusOK, util.OKResponse(r.Res))
		go func() {
			unityServer.Message <- "matrix " + s
		}()
	} else {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
	}
}

func forward(ctx *gin.Context) {
	r := MatrixReq{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	forwardServer.Message <- r.Res
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}

type OperationAll struct {
	Action string `json:"action"`
}

func operateAll(ctx *gin.Context) {
	r := OperationAll{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	go func() {
		unityServer.Message <- "operation allcontrol " + r.Action
	}()
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}

type Speed struct {
	AccValue int `json:"acc_value"`
	DecValue int `json:"dec_value"`
	MaxValue int `json:"max_value"`
}

func setSpeed(ctx *gin.Context) {
	r := Speed{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	go func() {
		unityServer.Message <- fmt.Sprintf("operation setspeed %d %d %d", r.AccValue, r.DecValue, r.MaxValue)
	}()
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}

type MotorActionReq struct {
	Id     string `json:"id"`
	Action string `json:"action"`
}

func motorAction(ctx *gin.Context) {
	r := MotorActionReq{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	go func() {
		unityServer.Message <- fmt.Sprintf("operation motorcontrol %s %s", r.Id, r.Action)
	}()
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}

type setWIFIReq struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func setWIFI(ctx *gin.Context) {
	r := setWIFIReq{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	go func() {
		unityServer.Message <- fmt.Sprintf("operation setwifi %s %s %s", r.Id, r.Name, r.Password)
	}()
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}

type setTCPReq struct {
	Id   string `json:"id"`
	IP   string `json:"ip"`
	Port string `json:"port"`
}

func setTCP(ctx *gin.Context) {
	r := setTCPReq{}
	if err := ctx.ShouldBindJSON(&r); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("para err: %s", err.Error())
		return
	}
	go func() {
		unityServer.Message <- fmt.Sprintf("operation settcp %s %s %s", r.Id, r.IP, r.Port)
	}()
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}
