package util

type CommonResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func OKResponse(data interface{}) *CommonResponse {
	return &CommonResponse{
		Code:    0,
		Message: "ok",
		Result:  data,
	}
}

func FailedResponse(code int, msg string) *CommonResponse {
	return &CommonResponse{
		Code:    code,
		Message: msg,
		Result:  nil,
	}
}
