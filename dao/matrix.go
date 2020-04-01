package dao

import (
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
	_, err := matrixEngine.Insert(m)
	if err != nil {
		log.Errorf("insert matrix err: %s", err.Error())
	} else {
		log.Infof("insert matrix success,id is %d", m.Id)
	}
	return err
}
func GetMatrix(id int64, command int) (matrix *model.Matrix, err error) {
	matrix = &model.Matrix{}
	var has bool
	if command == 0 {
		has, err = matrixEngine.Table("matrix").Asc("id").Where("id>?", id).Limit(1, 0).Get(matrix)
	} else {
		has, err = matrixEngine.Table("matrix").Desc("id").Where("id<?", id).Limit(1, 0).Get(matrix)
	}
	if !has {
		has, err = matrixEngine.Table("matrix").Where("id=?", id).Limit(1, 0).Get(matrix)
	}
	if err != nil {
		log.Errorf("get matrix err: %s", err.Error())
	}
	return
}
