package modules

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/log"
)

type Album struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	sysstamp     int64
	Sysdata      string   `json:"sysdate"`
	Public       int      `json:"public"`
	Visible      int      `json:"visible"`
	Downloadable int      `json:"downloadable"`
	Password     int      `json:"password"`
	ThumbUrls    []string `json:"thumbs"`
}

func (a *Album) FillThumbs(s *LycheeServer, conn *sql.DB) (err error) {
	a.ThumbUrls = make([]string, 0, 3)
	rows, err := conn.Query("SELECT thumbUrl FROM lychee_photos WHERE album = ? ORDER BY star DESC, "+s.Settings.SortingPhotos, a.Id)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var thumbUrl string
		rows.Scan(&thumbUrl)
		a.ThumbUrls = append(a.ThumbUrls, thumbUrl)
	}
	return
}

func (a *Album) PrepareData(s *LycheeServer, conn *sql.DB) (err error) {
	err = a.FillThumbs(s, conn)
	if err != nil {
		log.Error("%v", err)
		return
	}
	t := time.Unix(a.sysstamp, 0)
	a.Sysdata = t.Format("Jan 2006")
	return
}

func GetAlbumsAction(server *LycheeServer, c *gin.Context) {
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusBadRequest, "Get albums error")
	}
	defer conn.Close()
	albums, err := GetAlbums(server, conn)
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusBadRequest, "Get albums error")
	}
	for _, a := range albums {
		a.PrepareData(server, conn)
	}
	c.JSON(200, gin.H{"albums": albums,
		"num": len(albums),
		"smartalbums": gin.H{
			"unsorted": gin.H{"thumbs": make([]string, 0, 0), "num": 0},
			"starred":  gin.H{"thumbs": make([]string, 0, 0), "num": 0},
			"recent":   gin.H{"thumbs": make([]string, 0, 0), "num": 0},
			"public":   gin.H{"thumbs": make([]string, 0, 0), "num": 0},
		},
	})
}

func GetAlbums(server *LycheeServer, conn *sql.DB) (albums []*Album, err error) {
	albums = make([]*Album, 0, 10)
	query := "SELECT id, title, public, sysstamp, password FROM lychee_albums WHERE public = 1 AND visible <> 0 " + server.Settings.SortingAlbums
	log.Debug("Running query: " + query)
	rows, err := conn.Query(query)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		album := &Album{}
		err = rows.Scan(&album.Id, &album.Title, &album.Public, &album.sysstamp, &album.Password)
		if err != nil {
			return
		}
		albums = append(albums, album)
	}
	return
}
