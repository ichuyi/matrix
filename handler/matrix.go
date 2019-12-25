package handler

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"matrix/dao"
	"matrix/model"
	"matrix/socket"
	"matrix/util"
	"net/http"
)

var current int64 = 0

type Req struct {
	Res string `json:"res"`
}

func resource(ctx *gin.Context) {
	r := Req{}
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
		current = m.Id
		ctx.JSON(http.StatusOK, util.OKResponse(m.MatrixInfo))
		go func() {
			socket.Message <- m.MatrixInfo
		}()
	} else if r.Res == "backend" {
		m, err := dao.GetMatrix(current, 1)
		if err != nil {
			ctx.JSON(http.StatusOK, util.FailedResponse(GetMatrix, GetMatrixMsg))
			return
		}
		current = m.Id
		ctx.JSON(http.StatusOK, util.OKResponse(m.MatrixInfo))
		go func() {
			socket.Message <- m.MatrixInfo
		}()
	} else if util.CheckArr(r.Res) {
		m := &model.Matrix{
			MatrixInfo: r.Res,
		}
		err := dao.AddMatrix(m)
		if err != nil {
			ctx.JSON(http.StatusOK, util.FailedResponse(AddMatrix, AddMatrixMsg))
			return
		}
		current = m.Id
		ctx.JSON(http.StatusOK, util.OKResponse(r.Res))
		go func() {
			socket.Message <- r.Res
		}()
	} else {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
	}
}
