package modules

import (
	"github.com/fatedier/frp/utils/log"
	"github.com/gin-gonic/gin"
)

func ServeFunction(c *gin.Context) {
	functionName := c.PostForm("function")
	if functionName == "" {
		functionName = c.Query("function")
	}
	log.Debug("Running for function" + functionName)
	c.JSON(200, gin.H{"test": "test"})
}
