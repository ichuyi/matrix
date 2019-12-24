package model

import (
	"encoding/json"
	"errors"
	"matrix/util"
	"strconv"
	"strings"
	"time"
)

type Matrix struct {
	Id          int64
	CreatedTime time.Time `json:"created_time" xorm:"created"`
	MatrixInfo  string    `json:"matrix_info" xorm:"varchar(1000) notnull"`
	IsRead      int       `json:"is_read" xorm:"tinyint"`
}

type ReturnedMatrix struct {
	Id          int64     `json:"id"`
	CreatedTime time.Time `json:"created_time"`
	MatrixInfo  [][]int   `json:"matrix_info"`
	IsRead      int       `json:"is_read"`
}

func NewReturnedMatrix() *ReturnedMatrix {
	r := &ReturnedMatrix{
		Id:          0,
		CreatedTime: time.Time{},
		MatrixInfo:  nil,
		IsRead:      0,
	}
	r.MatrixInfo = make([][]int, util.ConfigInfo.Matrix.Width)
	for i := 0; i < util.ConfigInfo.Matrix.Width; i++ {
		r.MatrixInfo[i] = make([]int, util.ConfigInfo.Matrix.Length)
	}
	return r
}

func (returnedMatrix *ReturnedMatrix) ToMatrix() (matrix *Matrix, err error) {
	matrix = &Matrix{
		Id:          returnedMatrix.Id,
		CreatedTime: returnedMatrix.CreatedTime,
		IsRead:      returnedMatrix.IsRead,
	}
	s, err := json.Marshal(returnedMatrix.MatrixInfo)
	if err == nil {
		matrix.MatrixInfo = string(s)
	}
	return
}
func (returnedMatrix *ReturnedMatrix) Init(s string) error {
	in := strings.Split(s, " ")
	if len(in) != util.ConfigInfo.Matrix.Length*util.ConfigInfo.Matrix.Width {
		return errors.New("字符串长度错误")
	}
	for i := 0; i < util.ConfigInfo.Matrix.Width*util.ConfigInfo.Matrix.Length; i++ {
		n, err := strconv.Atoi(in[i])
		if err != nil {
			return errors.New("字符串解析错误")
		}
		returnedMatrix.MatrixInfo[i/util.ConfigInfo.Matrix.Length][i%util.ConfigInfo.Matrix.Length] = n
	}
	return nil
}

func (matrix *Matrix) ToReturnedMatrix() (r *ReturnedMatrix, err error) {
	r = &ReturnedMatrix{
		Id:          matrix.Id,
		CreatedTime: matrix.CreatedTime,
		IsRead:      matrix.IsRead,
	}
	r.MatrixInfo = make([][]int, util.ConfigInfo.Matrix.Width)
	for i := 0; i < util.ConfigInfo.Matrix.Width; i++ {
		r.MatrixInfo[i] = make([]int, util.ConfigInfo.Matrix.Length)
	}
	err = json.Unmarshal([]byte(matrix.MatrixInfo), &(r.MatrixInfo))
	return
}
