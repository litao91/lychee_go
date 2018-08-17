package modules

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/helper"
	"github.com/litao91/lychee_go/util/log"
)

type Album struct {
	Id           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	sysstamp     int64
	Sysdate      string   `json:"sysdate"`
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
	a.Sysdate = t.Format("Jan 2006")
	return
}

func GetAlbumsAction(server *LycheeServer, c *gin.Context) {
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, "Get albums error")
	}
	defer conn.Close()
	albums, err := GetAlbums(server, conn)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, "Get albums error")
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
	query := "SELECT id, title, public, sysstamp FROM lychee_albums WHERE visible <> 0 " + server.Settings.SortingAlbums
	log.Debug("Running query: " + query)
	rows, err := conn.Query(query)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		album := &Album{}
		err = rows.Scan(&album.Id, &album.Title, &album.Public, &album.sysstamp)
		if err != nil {
			return
		}
		albums = append(albums, album)
	}
	return
}

func GetAlbum(albumID int, conn *sql.DB) (album *Album, err error) {
	album = &Album{}
	query := "SELECT id, title, public, sysstamp FROM lychee_albums WHERE id = ? "
	err = conn.QueryRow(query, albumID).Scan(&album.Id, &album.Title, &album.Public, &album.sysstamp)
	return
}

func AddAlbumAction(server *LycheeServer, c *gin.Context) {
	title := c.PostForm("title")
	log.Info("Creating album with title " + title)
	if title == "" {
		c.String(http.StatusBadRequest, "title can't be empty")
		return
	}
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusInternalServerError, "Can't connect to DB")
		return
	}
	defer conn.Close()

	id := helper.GenerateID()
	sysstamp := time.Now().Unix()
	public := 0
	visible := 1

	query := "INSERT INTO lychee_albums (id, title, sysstamp, public, visible) VALUES (?, ?, ?, ?, ?)"

	_, err = conn.Exec(query, id, title, sysstamp, public, visible)
	if err != nil {
		c.String(http.StatusBadRequest, "Can't add album with title "+title)
		return
	}
	c.String(200, id)
}

func GetAlbumAction(server *LycheeServer, c *gin.Context) {
	albumID, err := strconv.Atoi(c.PostForm("albumID"))
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusBadRequest, "Invalid album id")
		return
	}
	log.Debug("Get Album detail of ID %d", albumID)

	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusInternalServerError, "Can't connect to DB")
		return
	}
	defer conn.Close()
	album, err := GetAlbum(albumID, conn)
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	album.PrepareData(server, conn)
	photos, err := LoadPhotosOfAlbum(albumID, conn)
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	c.JSON(200, gin.H{
		"id":           album.Id,
		"title":        album.Title,
		"public":       album.Public,
		"description":  album.Description,
		"visible":      album.Visible,
		"downloadable": album.Downloadable,
		"sysdate":      album.Sysdate,
		"password":     album.Password,
		"thumbs":       album.ThumbUrls,
		"content":      photos,
	})
}
