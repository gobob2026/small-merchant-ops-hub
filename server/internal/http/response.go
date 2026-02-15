package http

import "github.com/gin-gonic/gin"

type apiResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(200, apiResponse{
		Code: 200,
		Msg:  "ok",
		Data: data,
	})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(200, apiResponse{
		Code: code,
		Msg:  msg,
		Data: gin.H{},
	})
}
