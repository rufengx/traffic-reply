package httphandle

import (
	"net/http"
	"xtransform/app/common/json"
)

const (
	OK          = 10200
	BAD_REQUEST = 10400
	NOT_FOUND   = 10404
	CONFLICT    = 10409
)

var statusText = map[int]string{
	OK:          "ok",
	BAD_REQUEST: "bad request.",
	NOT_FOUND:   "not found.",
	CONFLICT:    "Conflict",
}

func responseMsgMapping(code int) string {
	return statusText[code]
}

type ResponseMsg struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func WriteJson(res http.ResponseWriter, code int) {
	WriteJsonRawData(res, code, responseMsgMapping(code), nil)
}

func WriteJsonRaw(res http.ResponseWriter, code int, msg string) {
	WriteJsonRawData(res, code, msg, nil)
}

func WriteJsonData(res http.ResponseWriter, code int, data interface{}) {
	WriteJsonRawData(res, code, responseMsgMapping(code), data)
}

func WriteJsonRawData(res http.ResponseWriter, code int, msg string, data interface{}) {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(http.StatusOK)
	r := ResponseMsg{
		Code:    code,
		Message: msg,
		Data:    data,
	}
	res.Write([]byte(json.JsonEncode(r, true, true, true)))
}
