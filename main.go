package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type LycheeServer struct {
	host     string
	port     int64
	basePath string
	engine   *gin.Engine
}

func NewServer(filePath string, port int64) (server *LycheeServer) {
	server = &LycheeServer{
		host:     "0.0.0.0",
		port:     port,
		basePath: filePath,
		engine:   gin.Default(),
	}
	return
}

func (server *LycheeServer) Init() (err error) {
}

func main() {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	s := NewServer(wd, 3334)
	err = s.Init()
	if err != nil {
	}
	s.Run()
}
