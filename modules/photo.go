package modules

import (
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	humanize "github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/helper"
	"github.com/litao91/lychee_go/util/log"
	"github.com/rwcarlsen/goexif/exif"
)

const ImageTypeJpg = 0

const PhotoSelectStmt string = `
SELECT id, title, description, url, tags,
public, type, width, height, size, iso, aperture, make, model,
shutter, focal, takestamp, star, thumbUrl, album, checksum, medium
FROM lychee_photos`

type Photo struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Tags        string `json:"tags"`
	Public      string `json:"public"`
	Type        string `json:"type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Size        string `json:"size"`
	Iso         string `json:"iso"`
	Aperture    string `json:"aperture"`
	Make        string `json:"make"`
	Model       string `json:"model"`
	Shutter     string `json:"shutter"`
	Focal       string `json:"focal"`
	Takestamp   string `json:"takestamp"`
	Takedate    string `json:"takedate"`
	Star        string `json:"star"`
	ThumbUrl    string `json:"thumbUrl"`
	Album       int    `json:"album"`
	Checksum    string `json:"checksum"`
	Medium      string `json:"medium"`

	idStr       string
	filename    string
	dataPath    string
	imagePath   string
	uploadPath  string
	mediumPath  string
	thumbPath   string
	thumb2xPath string
	tempPath    string
	img         image.Image
}

func NewPhoto(server *LycheeServer, imgPath string, filename string, idStr string) (photo *Photo, err error) {
	photo = &Photo{
		idStr:     idStr,
		dataPath:  server.dataPath,
		imagePath: imgPath,
		filename:  filename,
		Public:    "0",
	}

	photo.ID, err = strconv.ParseInt(photo.idStr, 10, 64)
	if err != nil {
		log.Error("%v", err)
		return
	}

	checksum, err := helper.HashFileSha1(imgPath)
	if err != nil {
		log.Error("%v", err)
		return
	}

	file, err := os.Open(imgPath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer file.Close()
	img, format, err := image.Decode(file)
	if err != nil {
		log.Error("%v", err)
	}

	photo.img = img
	photo.Type = format

	photo.Checksum = checksum

	photo.thumbPath = path.Join(server.thumbsDir, checksum+".jpg")
	photo.thumb2xPath = path.Join(server.thumbsDir, checksum+"@2x.jpg")
	photo.uploadPath = path.Join(server.uploadsDir, photo.idStr+"_"+filename)

	photo.mediumPath = path.Join(server.mediumDir, checksum+".jpg")

	return
}

func LoadPhotosOfAlbum(albumID int, conn *sql.DB) (photos []*Photo, err error) {
	query := PhotoSelectStmt + " WHERE album = ?"
	rows, err := conn.Query(query, albumID)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer rows.Close()
	photos = make([]*Photo, 0, 4)
	var r *Photo
	for rows.Next() {
		r, err = loadPhotoFromRow(rows)
		if err != nil {
			log.Error("%v", err)
			return
		}
		photos = append(photos, r)
	}
	return
}

func loadPhotoFromRow(row *sql.Rows) (r *Photo, err error) {
	r = &Photo{}
	err = row.Scan(&r.ID, &r.Title, &r.Description, &r.Url, &r.Tags, &r.Public, &r.Type, &r.Width, &r.Height,
		&r.Size, &r.Iso, &r.Aperture, &r.Make, &r.Model, &r.Shutter, &r.Focal, &r.Takestamp, &r.Star,
		&r.ThumbUrl, &r.Album, &r.Checksum, &r.Medium)
	return
}

func (photo *Photo) Exists(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM lychee_photos where checksum = ?", photo.Checksum).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func validateExtension(filename string) (string, bool) {
	isValid := strings.HasSuffix(strings.ToLower(filename), ".jpg")
	return "jpg", isValid
}

func GetPhotoAction(server *LycheeServer, c *gin.Context) {
	photoId := c.PostForm("photoID")
	log.Debug("ID: " + photoId)
	r := &Photo{}
	query := PhotoSelectStmt + " WHERE id = ?"
	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	err = conn.QueryRow(query, photoId).Scan(&r.ID, &r.Title, &r.Description, &r.Url, &r.Tags, &r.Public, &r.Type, &r.Width, &r.Height,
		&r.Size, &r.Iso, &r.Aperture, &r.Make, &r.Model, &r.Shutter, &r.Focal, &r.Takestamp, &r.Star,
		&r.ThumbUrl, &r.Album, &r.Checksum, &r.Medium)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	ts, e := strconv.ParseInt(r.Takestamp, 10, 64)
	if e == nil {
		t := time.Unix(ts, 0)
		r.Takedate = t.Format(time.RFC3339)
	} else {
		r.Takedate = r.Takestamp
	}

	c.JSON(200, r)

}

func SetPhotoAlbumAction(server *LycheeServer, c *gin.Context) {
	conn, err := server.GetDBConnection()
	if err != nil {
		c.JSON(500, fmt.Sprintf("%v", err))
		return
	}
	defer conn.Close()
	photoIds := c.PostForm("photoIDs")
	albumId := c.PostForm("albumID")
	if err != nil {
		log.Error("%v", err)
		c.JSON(500, fmt.Sprintf("%v", err))
		return
	}
	_, err = conn.Exec(fmt.Sprintf("Update lychee_photos SET album = ? WHERE ID IN (%s)", photoIds), albumId)
	if err != nil {
		log.Error("%v", err)
		c.JSON(200, false)
		return
	}
	c.JSON(200, true)
	return
}

func SetStar(db *sql.DB, photoIDs string) (interface{}, error) {
	log.Debug("Star for %s", photoIDs)
	query := fmt.Sprintf("SELECT id, star FROM lychee_photos WHERE id in (%s)", photoIDs)
	rows, err := db.Query(query)
	if err != nil {
		log.Error("%v", err)
		return false, err
	}
	defer rows.Close()

	idStar := [][]int{}
	for rows.Next() {
		var id, star int
		rows.Scan(&id, &star)
		log.Debug("%d - %d", id, star)
		idStar = append(idStar, []int{id, star})
	}
	tx, err := db.Begin()
	if err != nil {
		log.Error("%v", err)
		return false, err
	}
	for _, i := range idStar {
		_, err := tx.Exec("UPDATE lychee_photos SET star = ? WHERE id = ?", 1-i[1], i[0])
		if err != nil {
			log.Error("%v", err)
			tx.Rollback()
			return false, err
		}
	}

	tx.Commit()
	return true, nil
}

func SetPhotoTitle(db *sql.DB, photoIDs string, title string) (interface{}, error) {
	_, err := db.Exec(fmt.Sprintf("UPDATE lychee_photos SET title = ? WHERE id in (%s)", photoIDs), title)
	if err != nil {
		return false, err
	}
	return true, nil
}

func SetPhotoDescription(db *sql.DB, photoIDs string, description string) (interface{}, error) {
	log.Debug("Set description of %s to %s", photoIDs, description)
	_, err := db.Exec(fmt.Sprintf("UPDATE lychee_photos SET description = ? WHERE id in (%s)", photoIDs), description)
	if err != nil {
		return false, err
	}
	return true, nil
}

func SetPhotoTags(db *sql.DB, photoIDs string, tags string) (interface{}, error) {
	_, err := db.Exec(fmt.Sprintf("UPDATE lychee_photos SET tags = ? WHERE id in (%s)", photoIDs), tags)
	if err != nil {
		return false, err
	}
	return true, nil
}

func DeletePhotoAction(server *LycheeServer, c *gin.Context) {
	photoIDs := c.PostForm("photoIDs")
	db, err := server.GetDBConnection()
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	query := fmt.Sprintf("SELECT url, thumbUrl, medium FROM lychee_photos WHERE id in (%s)", photoIDs)
	rows, err := db.Query(query)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		return
	}
	defer rows.Close()
	for rows.Next() {
		var url, thumbUrl, medium string
		err := rows.Scan(&url, &thumbUrl, &medium)
		if err != nil {
			log.Error("%v", err)
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("%v", err))
			return
		}
		img := path.Join(server.dataPath, url)
		thumb := path.Join(server.dataPath, thumbUrl)
		splitted := strings.Split(thumb, ".")
		thumb2x := splitted[0] + "@2x" + splitted[1]
		m := path.Join(server.dataPath, medium)
		log.Debug("Deleting %s, %s, %s, %s", img, thumb, thumb2x, m)
		log.Debug("DOn't do real deletion for now")
	}

	query = fmt.Sprintf("DELETE from lychee_photos WHERE id in (%s)", photoIDs)
	_, err = db.Exec(query)
	if err != nil {
		log.Error("%v", err)
		c.JSON(500, false)
		return
	}

	c.JSON(200, true)
}

func UploadAction(server *LycheeServer, c *gin.Context) {
	albumId, err := strconv.Atoi(c.PostForm("albumID"))
	log.Debug("Uploading image to album: %d", albumId)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	file, err := c.FormFile("0")
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	_, isValid := validateExtension(file.Filename)
	if !isValid {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("Not a valid image file extension"))
		return
	}
	log.Debug("Uploading file %s", file.Filename)

	id := helper.GenerateID()
	tmpFilepath := path.Join(server.tmpDir, id)
	if err := c.SaveUploadedFile(file, tmpFilepath); err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}
	photo, err := NewPhoto(server, tmpFilepath, file.Filename, id)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
	}
	photo.Album = albumId

	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	defer conn.Close()
	err = photo.SavePhoto(conn, true)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	c.JSON(http.StatusOK, id)
}

func (photo *Photo) GenPhotoExif() (err error) {
	photo.Height = photo.img.Bounds().Size().Y
	photo.Width = photo.img.Bounds().Size().X
	fi, e := os.Stat(photo.imagePath)
	if e != nil {
		log.Error("%v", e)
		return e
	}
	// get the size
	photo.Size = humanize.Bytes(uint64(fi.Size()))
	f, err := os.Open(photo.imagePath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer f.Close()

	x, e := exif.Decode(f)
	if e != nil {
		log.Error("%v", e)
		return nil
	}

	model, err := x.Get(exif.Model)
	if err != nil {
		log.Error("Model: %v", err)
	} else {
		photo.Model, _ = model.StringVal()
	}

	iso, e := x.Get(exif.ISOSpeedRatings)
	if e != nil {
		log.Error("ISO: %v", e)
	} else {
		photo.Iso = iso.String()
		log.Debug("ISO: " + photo.Iso)
	}

	aperture, e := x.Get(exif.FNumber)
	if e != nil {
		log.Error("%v", e)
	} else {
		numer, denom, e := aperture.Rat2(0)
		if e != nil {
			log.Error("%v", e)
		} else {
			photo.Aperture = fmt.Sprintf("%.1f", float64(numer)/float64(denom))
			log.Info("Aperture " + photo.Aperture)
		}
	}

	make, e := x.Get(exif.Make)
	if e != nil {
		log.Error("%v", e)
	} else {
		photo.Make, e = make.StringVal()
		if e != nil {
			log.Error("%v", e)
		}
		log.Info("Make " + photo.Make)
	}

	shutter, e := x.Get(exif.ExposureTime)
	if e != nil {
		log.Error("%v", e)
	} else {
		photo.Shutter = fmt.Sprintf("%s s", strings.Trim(shutter.String(), "\""))
		log.Info("Shutter " + photo.Shutter)
	}

	focal, e := x.Get(exif.FocalLength)
	if e != nil {
		log.Error("%v", e)
	} else {
		numer, denom, e := focal.Rat2(0)
		if e != nil {
			log.Error("%v", e)
		}
		photo.Focal = fmt.Sprintf("%v mm", numer/denom)
		log.Info("Focal " + photo.Focal)
	}

	ts, e := x.DateTime()
	if e != nil {
		log.Error("%v", e)
	} else {
		photo.Takestamp = fmt.Sprintf("%v", ts.Unix())
		log.Info("Takestamp " + photo.Takestamp)
	}

	return nil
}

func (photo *Photo) CopyToUpload() (err error) {
	from, err := os.Open(photo.imagePath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer from.Close()
	to, err := os.Create(photo.uploadPath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	return
}

func (photo *Photo) SavePhotoMeta(db *sql.DB) error {
	_, err := db.Exec(`
		 INSERT INTO lychee_photos (id, title, url, description, tags, type, width, height, size, iso, aperture, make, model, shutter, focal, takestamp, thumbUrl, album, public, star, checksum, medium) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 `, photo.ID, photo.Title, photo.Url, photo.Description, photo.Tags, photo.Type, photo.Width, photo.Height,
		photo.Size, photo.Iso, photo.Aperture, photo.Make, photo.Model, photo.Shutter, photo.Focal, photo.Takestamp, photo.ThumbUrl, photo.Album, photo.Public, photo.Star, photo.Checksum, photo.Medium)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	return nil
}

func (photo *Photo) SavePhoto(db *sql.DB, copyToUpload bool) (err error) {
	exists, err := photo.Exists(db)
	if err != nil {
		log.Error("%v", err)
		return err
	}

	if exists {
		return fmt.Errorf("Photo exists")
	}

	if copyToUpload {
		err = photo.CopyToUpload()
		if err != nil {
			log.Error("%v", err)
			return
		}
		photo.Url, err = filepath.Rel(photo.dataPath, photo.uploadPath)
		if err != nil {
			log.Error("%v", err)
			return
		}
	} else {
		photo.Url, err = filepath.Rel(photo.dataPath, photo.imagePath)
		if err != nil {
			log.Error("%v", err)
			return
		}
	}
	photo.createMedium()

	err = photo.createThumb()
	if err != nil {
		log.Error("%v", err)
		return
	}
	err = photo.GenPhotoExif()
	if err != nil {
		log.Error("%v", err)
		return
	}

	err = photo.SavePhotoMeta(db)
	if err != nil {
		log.Error("%v", err)
		return
	}

	return
}

func (photo *Photo) createThumb() error {
	thumb := imaging.Thumbnail(photo.img, 180, 180, imaging.Lanczos)
	thumb2x := imaging.Thumbnail(photo.img, 360, 360, imaging.Lanczos)
	out, err := os.Create(photo.thumbPath)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	defer out.Close()
	err = jpeg.Encode(out, thumb, nil)
	if err != nil {
		log.Error("%v")
		return err
	}
	out2x, err := os.Create(photo.thumb2xPath)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	defer out2x.Close()
	err = jpeg.Encode(out2x, thumb2x, nil)
	if err != nil {
		log.Error("%v")
		return err
	}

	photo.ThumbUrl, err = filepath.Rel(photo.dataPath, photo.thumbPath)
	return err
}

func (photo *Photo) createMedium() {
	if helper.DoesFileExists(photo.mediumPath) {
		log.Info("Medium file %s exists, continue", photo.mediumPath)
		return
	}
	height := photo.img.Bounds().Size().Y
	width := photo.img.Bounds().Size().X
	if height <= 1920 && width <= 1920 {
		photo.Medium = ""
		return
	}
	var newWidth int = 1920
	if width < height {
		newWidth = 1080
	}
	m := imaging.Resize(photo.img, newWidth, 0, imaging.Lanczos)
	out, err := os.Create(photo.mediumPath)
	if err != nil {
		log.Error("%v", err)
		photo.Medium = ""
		return
	}
	defer out.Close()
	err = jpeg.Encode(out, m, nil)
	if err == nil {
		photo.Medium, _ = filepath.Rel(photo.dataPath, photo.mediumPath)
	} else {
		log.Error("%v", err)
		photo.Medium = ""
	}
	return
}
