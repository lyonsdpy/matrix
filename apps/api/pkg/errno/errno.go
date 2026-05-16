/**
 * @Author: DPY
 * @Description:
 * @File:  errno
 * @Version: 1.0.0
 * @Date: 2021/12/13 17:18
 */

package errno

import (
	"encoding/json"
	"matrix/api/pkg/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Err struct {
	Code int    `json:"code,omitempty"`
	Err  string `json:"error,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func (e Err) ErrLog() {
	var r []zap.Field
	var msg string
	if e.Code != 0 {
		r = append(r, zapcore.Field{Key: "code", Type: zapcore.Int64Type, Integer: int64(e.Code)})
	}
	if e.Err != "" {
		r = append(r, zapcore.Field{Key: "error", Type: zapcore.StringType, String: e.Err})
	}
	if e.Msg != "" {
		msg = e.Msg
	}
	log.Log.Error(msg, r...)
}

func (e Err) Error() string {
	errBytes, _ := json.Marshal(e)
	return string(errBytes)
}

func (e Err) WithErr(err error) Err {
	if err != nil {
		e.Err = err.Error()
	}
	return e
}

func (e Err) WithString(err string) Err {
	e.Err = err
	return e
}

func (e Err) WithMsg(msg string) Err {
	e.Msg = msg
	return e
}

func New(code int, msg string) Err {
	return Err{Code: code, Msg: msg}
}
