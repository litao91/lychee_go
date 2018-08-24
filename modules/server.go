package modules

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/helper"
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

	uploadsDir  string
	mediumDir   string
	thumbsDir   string
	tmpDir      string
	staticPaths []string
}

type LycheeFunc func(*LycheeServer, *gin.Context)
type ActionFunc func(*sql.DB, string) (interface{}, error)
type ActionFuncTwoArg func(*sql.DB, string, string) (interface{}, error)

func ActionToLycheeFunc(action ActionFunc, arg string) LycheeFunc {
	return func(server *LycheeServer, c *gin.Context) {
		conn, err := server.GetDBConnection()
		if err != nil {
			log.Error("%v", err)
			c.JSON(500, fmt.Sprintf("%v", err))
		}
		defer conn.Close()
		r, err := action(conn, c.PostForm(arg))
		if err != nil {
			log.Error("%v", err)
			c.JSON(500, fmt.Sprintf("%v", err))
		}
		c.JSON(200, r)
	}
}

func ActionToLycheeFuncTwoArg(action ActionFuncTwoArg, arg1 string, arg2 string) LycheeFunc {
	return func(server *LycheeServer, c *gin.Context) {
		conn, err := server.GetDBConnection()
		if err != nil {
			log.Error("%v", err)
			c.JSON(500, fmt.Sprintf("%v", err))
		}
		defer conn.Close()
		r, err := action(conn, c.PostForm(arg1), c.PostForm(arg2))
		if err != nil {
			log.Error("%v", err)
			c.JSON(500, fmt.Sprintf("%v", err))
		}
		c.JSON(200, r)
	}
}

var lycheeFuncMap map[string]LycheeFunc = map[string]LycheeFunc{
	"Session::init":         InitAction,
	"Session::login":        LoginAction,
	"Albums::get":           GetAlbumsAction,
	"Album::add":            AddAlbumAction,
	"Album::get":            GetAlbumAction,
	"Album::setTitle":       ActionToLycheeFuncTwoArg(SetAlbumTitle, "albumIDs", "title"),
	"Album::setDescription": ActionToLycheeFuncTwoArg(SetAlbumDescription, "albumIDs", "description"),
	"Album::delete":         ActionToLycheeFunc(DeleteAlbum, "albumIDs"),
	"Photo::add":            UploadAction,
	"Photo::get":            GetPhotoAction,
	"Photo::setAlbum":       SetPhotoAlbumAction,
	"Photo::setStar":        ActionToLycheeFunc(SetStar, "photoIDs"),
	"Photo::setTitle":       ActionToLycheeFuncTwoArg(SetPhotoTitle, "photoIDs", "title"),
	"Photo::setDescription": ActionToLycheeFuncTwoArg(SetPhotoDescription, "photoID", "description"),
	"Photo::setTags":        ActionToLycheeFuncTwoArg(SetPhotoTags, "photoIDs", "tags"),
}

func (server *LycheeServer) GetDBConnection() (db *sql.DB, err error) {
	db, err = server.db.GetConnection()
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

func (server *LycheeServer) initSessions() (err error) {
	// init session
	store := cookie.NewStore([]byte("lychee"))
	server.router.Use(sessions.Sessions("lychee", store))
	return nil
}

func (server *LycheeServer) initStaticDirectories() {
	server.router.Use(static.Serve("/dist", static.LocalFile(path.Join(server.basePath, "dist"), false)))
	server.router.Use(static.Serve("/src", static.LocalFile(path.Join(server.basePath, "src"), false)))
	for _, i := range server.staticPaths {
		s := strings.Split(i, "/")
		p := "/" + s[len(s)-1]
		log.Debug("Serving " + p)
		server.router.Use(static.Serve(p, static.LocalFile(i, false)))
	}
}

func (server *LycheeServer) prepareDataDirs() {
	server.uploadsDir = path.Join(server.dataPath, "uploads")
	server.thumbsDir = path.Join(server.dataPath, "thumbs")
	server.mediumDir = path.Join(server.dataPath, "medium")
	helper.CreateDirIfNotExists(server.uploadsDir)
	helper.CreateDirIfNotExists(server.thumbsDir)
	helper.CreateDirIfNotExists(server.mediumDir)
	server.tmpDir = path.Join(server.dataPath, "tmp")
	helper.CreateDirIfNotExists(server.tmpDir)
	server.staticPaths = []string{server.uploadsDir, server.thumbsDir, server.mediumDir, path.Join(server.dataPath, "Pictures"), path.Join(server.dataPath, "pictures")}
	return
}

func (server *LycheeServer) Init() (err error) {
	server.db.InitDb()
	server.initSessions()

	// serve the index file for root
	server.router.GET("/", server.ServeFile("index.html"))
	server.router.POST("/php/index.php", server.ServeFunction)
	server.router.GET("/php/index.php", server.ServeFunction)
	server.prepareDataDirs()
	server.initStaticDirectories()
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
