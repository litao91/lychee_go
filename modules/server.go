package modules

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/fatedier/frp/utils/log"
	"github.com/gin-gonic/gin"
)

type LycheeServer struct {
	host     string
	port     int64
	basePath string
	router   *gin.Engine
}

func (server *LycheeServer) ServeFile(relativePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fullPath := path.Join(server.basePath, relativePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			c.Writer.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(c.Writer, "file %s not found", fullPath)
			return
		}
		log.Debug("Serving file: %s", fullPath)
		http.ServeFile(c.Writer, c.Request, fullPath)
		return
	}

}

func (server *LycheeServer) ServeFunction(c *gin.Context) {
	functionName := c.PostForm("function")
	if functionName == "" {
		functionName = c.Query("function")
	}
	log.Debug("Running for function" + functionName)
	c.JSON(200, gin.H{"test": "test"})
}

func (server *LycheeServer) Init() (err error) {
	// serve the index file for root
	server.router.GET("/", server.ServeFile("index.html"))
	server.router.GET("/dist/main.js", server.ServeFile("dist/main.js"))
	server.router.GET("/dist/main.css", server.ServeFile("dist/main.css"))
	server.router.GET("/dist/view.js", server.ServeFile("dist/view.js"))
	server.router.POST("/php/index.php", server.ServeFunction)
	server.router.GET("/php/index.php", server.ServeFunction)
	return
}

func (server *LycheeServer) Run() {
	server.router.Run(fmt.Sprintf("%s:%d", server.host, server.port))
}

func NewServer(filePath string, port int64) (server *LycheeServer) {
	server = &LycheeServer{
		host:     "0.0.0.0",
		port:     port,
		basePath: filePath,
		router:   gin.Default(),
	}
	return
}
