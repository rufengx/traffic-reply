package controller

import "github.com/gin-gonic/gin"

func ScheduleController(c *gin.Context) {
	c.GetPostForm("http_input_config")
}
