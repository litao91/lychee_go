package modules

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/log"
)

func getSettings(server *LycheeServer) (settings *Settings, err error) {
	settings = &Settings{
		ThumbQuality:    "90",
		CheckForUpdates: "1",
		SortingPhotos:   "ORDER BY id DESC",
		DropboxKey:      "",
		Version:         "030100",
		Imagick:         "1",
		Medium:          "1",
		SortingAlbums:   "ORDER BY id DESC",
		SkipDuplicates:  "0",
		Location:        "",
		Login:           true,
	}
	return
}

// InitAction hardcode for now
func InitAction(server *LycheeServer, c *gin.Context) {
	log.Debug("Running Init Action")
	settings, err := getSettings(server)
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusBadRequest, "Getting config error")
		return
	}
	server.Settings = settings

	c.JSON(200, gin.H{"config": settings, "status": 2})
}

func LoginAction(server *LycheeServer, c *gin.Context) {
}
