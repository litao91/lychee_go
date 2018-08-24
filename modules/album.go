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

func GetSmartAlbums(s *LycheeServer, conn *sql.DB) (r map[string]map[string]interface{}, err error) {
	r = make(map[string]map[string]interface{})
	// unsorted
	unsorted := make(map[string]interface{})
	unsortedThumbs := make([]string, 0, 0)
	unsortedRows, err := conn.Query("SELECT thumbUrl FROM lychee_photos WHERE album = 0 ORDER BY " + s.Settings.SortingPhotos)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer unsortedRows.Close()
	for unsortedRows.Next() {
		var thumbUrl string
		unsortedRows.Scan(&thumbUrl)
		unsortedThumbs = append(unsortedThumbs, thumbUrl)
	}
	unsorted["thumbs"] = unsortedThumbs
	unsorted["num"] = len(unsortedThumbs)
	r["unsorted"] = unsorted

	// starred
	starred := make(map[string]interface{})
	starredThumbs := make([]string, 0, 0)
	starredRows, err := conn.Query("SELECT thumbUrl FROM lychee_photos WHERE star = 1 ORDER BY " + s.Settings.SortingPhotos)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer starredRows.Close()
	for starredRows.Next() {
		var thumbUrl string
		starredRows.Scan(&thumbUrl)
		starredThumbs = append(starredThumbs, thumbUrl)
	}

	starred["thumbs"] = starredThumbs
	starred["num"] = len(starredThumbs)
	r["starred"] = starred

	publicThumbs := make([]string, 0, 0)
	publicRows, err := conn.Query("SELECT thumbUrl From lychee_photos WHERE public = 1 ORDER BY " + s.Settings.SortingPhotos)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer publicRows.Close()

	r["public"] = gin.H{
		"thumbs": publicThumbs,
		"num":    len(publicThumbs),
	}

	recentThumbs := make([]string, 0, 0)
	now := time.Now().Unix() - 24*60*60
	recentRows, err := conn.Query("SELECT thumbUrl FROM lychee_photos WHERE id > ?", now)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer recentRows.Close()
	for recentRows.Next() {
		var thumbUrl string
		recentRows.Scan(&thumbUrl)
		recentThumbs = append(recentThumbs, thumbUrl)
	}

	r["recent"] = gin.H{
		"thumbs": recentThumbs,
		"num":    len(recentThumbs),
	}

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
	smartAlbums, err := GetSmartAlbums(server, conn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%s", err))
		return
	}
	c.JSON(200, gin.H{"albums": albums,
		"num":         len(albums),
		"smartalbums": smartAlbums,
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

func genPhotoMap(photos []*Photo) map[int64]map[string]interface{} {
	var photoMap map[int64]map[string]interface{} = make(map[int64]map[string]interface{})
	for i, p := range photos {
		next := photos[(i+1)%len(photos)].ID
		prev := photos[((i-1)+len(photos))%len(photos)].ID
		m := map[string]interface{}{
			"id":            p.ID,
			"title":         p.Title,
			"tags":          p.Tags,
			"public":        p.Public,
			"star":          p.Star,
			"album":         strconv.Itoa(p.Album),
			"thumbUrl":      p.ThumbUrl,
			"url":           p.Url,
			"previousPhoto": strconv.FormatInt(prev, 10),
			"nextPhoto":     strconv.FormatInt(next, 10),
		}
		if p.Takestamp != "" {
			m["cameraDate"] = "1"
			m["sysdate"] = p.Takestamp
		} else {
			m["cameraDate"] = "0"
			t := time.Unix(p.ID, 0)
			m["sysdate"] = t.Format("Jan 2006")
		}
		photoMap[p.ID] = m
	}
	return photoMap
}

func GetUserAlbum(albumIDStr string, conn *sql.DB, server *LycheeServer, c *gin.Context) {
	albumID, err := strconv.Atoi(albumIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
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
	photoMap := genPhotoMap(photos)
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
		"content":      photoMap,
	})
}

func GetSmartAlbum(albumID string, conn *sql.DB, server *LycheeServer, c *gin.Context) {
	var query string
	r := map[string]interface{}{}
	switch id := albumID; id {
	case "f":
		r["public"] = "0"
		query = PhotoSelectStmt + " WHERE star = 1"
	case "s":
		r["public"] = "0"
		query = PhotoSelectStmt + " WHERE public = 1"
	case "r":
		r["public"] = "0"
		ts := time.Now().Unix() - 24*60*60
		query = PhotoSelectStmt + " WHERE id > " + fmt.Sprintf("%d", ts)
	case "0":
		r["public"] = "0"
		query = PhotoSelectStmt + " WHERE album = 0"
	}
	query = query + " ORDER BY " + server.Settings.SortingPhotos
	rows, err := conn.Query(query)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	defer rows.Close()
	var photos []*Photo = make([]*Photo, 0, 0)
	for rows.Next() {
		p, err := loadPhotoFromRow(rows)
		if err != nil {
			log.Error("%v", err)
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
			return
		}
		photos = append(photos, p)
	}
	if len(photos) == 0 {
		c.JSON(200, gin.H{
			"content": false,
		})
		return
	}
	photoMap := genPhotoMap(photos)
	c.JSON(200, gin.H{
		"content": photoMap,
		"id":      albumID,
		"num":     len(photos),
	})

}

func GetAlbumAction(server *LycheeServer, c *gin.Context) {
	albumID := c.PostForm("albumID")
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.String(http.StatusInternalServerError, "Can't connect to DB")
		return
	}

	defer conn.Close()
	if len(albumID) > 2 {
		GetUserAlbum(albumID, conn, server, c)
	} else {
		GetSmartAlbum(albumID, conn, server, c)
	}
}
