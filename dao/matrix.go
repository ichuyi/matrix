package dao

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	log "github.com/sirupsen/logrus"
	"matrix/model"
	"matrix/util"
	"xorm.io/core"
)

var matrixEngine *xorm.Engine

func init() {
	connect := fmt.Sprintf("%s:%s@(%s:%s)/%s?charset=utf8", util.ConfigInfo.MySQL.User, util.ConfigInfo.MySQL.Password, util.ConfigInfo.MySQL.Host, util.ConfigInfo.MySQL.Port, util.ConfigInfo.MySQL.Database)
	var err error
	matrixEngine, err = xorm.NewEngine("mysql", connect)
	if err != nil {
		log.Fatalf(err.Error())
	}
	matrixEngine.ShowSQL(true)
	matrixEngine.Logger().SetLevel(core.LOG_DEBUG)
	err = matrixEngine.Ping()
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("success to connect to MySQL,connect info :", connect)
	err = matrixEngine.Sync2(new(model.Matrix))
	if err != nil {
		log.Errorf(err.Error())
	}
}

func AddMatrix(m *model.Matrix) error {
	id, err := matrixEngine.Insert(m)
	if err != nil {
		log.Errorf("insert matrix err: %s", err.Error())
	} else {
		log.Infof("insert matrix success,id is %d", id)
	}
	return err
}
func GetMatrix() (matrix *model.Matrix, err error) {
	matrix = &model.Matrix{}
	has, err := matrixEngine.Table("matrix").Asc("id").Where("is_read=?", 0).Limit(1, 0).Get(matrix)
	if !has {
		err = errors.New("矩阵已被全部读完")
	}
	if err != nil {
		log.Errorf("get matrix err: %s", err.Error())
		return
	}
	updateMatrix := &model.Matrix{
		IsRead: 1,
	}
	err = update(updateMatrix, matrix.Id)
	return
}
func update(matrix *model.Matrix, id int64) error {
	_, err := matrixEngine.Where("id=?", id).Update(matrix)
	if err != nil {
		log.Errorf("update matrix err: %s", err.Error())
	} else {
		log.Infof("update matrix success,id is %d", id)
	}
	return err
}
