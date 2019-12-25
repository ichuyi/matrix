package model

import (
	"time"
)

type Matrix struct {
	Id          int64
	CreatedTime time.Time `json:"created_time" xorm:"created"`
	MatrixInfo  string    `json:"matrix_info" xorm:"varchar(1000) notnull"`
}
