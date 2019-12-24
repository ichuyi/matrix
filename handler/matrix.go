package handler

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"matrix/dao"
	"matrix/model"
	"matrix/util"
	"net/http"
)

func addMatrix(ctx *gin.Context) {
	r := model.NewReturnedMatrix()
	input := ctx.PostForm("arr")
	if err := r.Init(input); err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(ParaError, ParaErrorMsg))
		log.Errorf("parse para err: %s", err.Error())
		return
	}
	m, err := r.ToMatrix()
	if err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(InternalError, InternalErrorMsg))
		log.Errorf("err: %s", err.Error())
		return
	}
	err = dao.AddMatrix(m)
	if err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(AddMatrix, AddMatrixMsg))
		return
	}
	ctx.JSON(http.StatusOK, util.OKResponse(nil))
}
func getMatrix(ctx *gin.Context) {
	m, err := dao.GetMatrix()
	if err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(GetMatrix, GetMatrixMsg))
		return
	}
	r, err := m.ToReturnedMatrix()
	if err != nil {
		ctx.JSON(http.StatusOK, util.FailedResponse(InternalError, InternalErrorMsg))
		log.Errorf("err: %s", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, util.OKResponse(r.MatrixInfo))
}
