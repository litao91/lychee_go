package modules

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/log"
)

type Settings struct {
	ThumbQuality    string `json:"thumbQuality"`
	CheckForUpdates string `json:"checkForUpdates"`
	SortingPhotos   string `json:"sortingPhotos"`
	DropboxKey      string `json:"dropboxKey"`
	Version         string `json:"version"`
	Imagick         string `json:"imagick"`
	Medium          string `json:"medium"`
	SortingAlbums   string `json:"sortingAlbums"`
	SkipDuplicates  string `json:"skipDuplicates"`
	Location        string `json:"location"`
	Login           bool   `json:"login"`
}

type LycheeServer struct {
	host     string
	port     int64
	basePath string
	dataPath string
	router   *gin.Engine
	db       *LycheeDb
	Settings *Settings
}

type LycheeFunc func(*LycheeServer, *gin.Context)

var lycheeFuncMap map[string]LycheeFunc = map[string]LycheeFunc{
	"Session::init":  InitAction,
	"Session::login": LoginAction,
	"Albums::get":    GetAlbumsAction,
	"Album::add":     AddAlbumAction,
	"Album::get":     GetAlbumAction,
	"Photo::add":     UploadAction,
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
	log.Debug("Running for function: %s", functionName)
	f, ok := lycheeFuncMap[functionName]
	if !ok {
		c.String(http.StatusBadRequest, "Can't find function "+functionName)
		return
	}
	f(server, c)
}

func (server *LycheeServer) InitSessions() (err error) {
	// init session
	store := cookie.NewStore([]byte("lychee"))
	server.router.Use(sessions.Sessions("lychee", store))
	return nil
}

func (server *LycheeServer) Init() (err error) {
	server.db.InitDb()
	server.InitSessions()
	server.router.Use(static.Serve("/data", static.LocalFile(server.dataPath, false)))
	server.router.Use(static.Serve("/dist", static.LocalFile(path.Join(server.basePath, "dist"), false)))
	server.router.Use(static.Serve("/src", static.LocalFile(path.Join(server.basePath, "src"), false)))

	// serve the index file for root
	server.router.GET("/", server.ServeFile("index.html"))
	server.router.POST("/php/index.php", server.ServeFunction)
	server.router.GET("/php/index.php", server.ServeFunction)
	return
}

func (server *LycheeServer) Run() {
	server.router.Run(fmt.Sprintf("%s:%d", server.host, server.port))
}

func NewServer(filePath string, dataPath string, port int64) (server *LycheeServer) {
	server = &LycheeServer{
		host:     "0.0.0.0",
		port:     port,
		basePath: filePath,
		router:   gin.Default(),
		dataPath: dataPath,
		db:       NewLycheeDb(path.Join(dataPath, "mainlib.db")),
	}
	return
}
