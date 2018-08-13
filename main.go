package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/modules"
	"github.com/litao91/lychee_go/util/log"
)

type LycheeServer struct {
	host     string
	port     int64
	basePath string
	router   *gin.Engine
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

func (server *LycheeServer) Init() (err error) {
	// serve the index file for root
	server.router.GET("/", server.ServeFile("index.html"))
	server.router.GET("/dist/main.js", server.ServeFile("dist/main.js"))
	server.router.GET("/dist/main.css", server.ServeFile("dist/main.css"))
	server.router.GET("/dist/view.js", server.ServeFile("dist/view.js"))
	server.router.POST("/php/index.php", modules.ServeFunction)
	server.router.GET("/php/index.php", modules.ServeFunction)
	return
}

func (server *LycheeServer) Run() {
	server.router.Run(fmt.Sprintf("%s:%d", server.host, server.port))
}

func main() {
	wd, err := filepath.Abs(filepath.Dir(os.Args[1]))
	log.Debug("Working directory: %s", wd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	s := NewServer(wd, 3334)
	err = s.Init()
	if err != nil {
		log.Error("%v", err)
	}
	s.Run()
}
