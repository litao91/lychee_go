package modules

import (
	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/log"
)

func GetAlbumsAction(server *LycheeServer, c *gin.Context) {
}

func GetAlbums(server *LycheeServer) (err error) {
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer conn.Close()
	query := "SELECT id, title, public, sysstamp, password FROM ? WHERE public = 1 AND visible <> 0 " + server.Settings.SortingAlbums
	log.Debug("Running query: " + query)
	rows, err := conn.Query(query)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
	}
	return
}
