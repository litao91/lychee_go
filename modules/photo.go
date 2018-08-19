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

	"github.com/disintegration/imaging"
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
	Public      int    `json:"public"`
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
	Star        int    `json:"star"`
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
		Public:    0,
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

	photo.mediumPath = path.Join(server.mediumDir, checksum+".jpg")

	photo.uploadPath = path.Join(server.uploadsDir, photo.idStr+"_"+filename)

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
	return count > 10, nil
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

	c.JSON(200, r)

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
	photo.Size = photo.img.Bounds().Size().String()

	f, err := os.Open(photo.imagePath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer f.Close()

	x, err := exif.Decode(f)

	model, _ := x.Get(exif.Model)
	photo.Model, _ = model.StringVal()

	iso, _ := x.Get(exif.ISOSpeedRatings)
	photo.Iso, _ = iso.StringVal()

	aperture, _ := x.Get(exif.ApertureValue)
	photo.Aperture, _ = aperture.StringVal()

	make, _ := x.Get(exif.Make)
	photo.Make, _ = make.StringVal()

	shutter, _ := x.Get(exif.ShutterSpeedValue)
	photo.Shutter, _ = shutter.StringVal()

	focal, _ := x.Get(exif.FocalLength)
	photo.Focal, _ = focal.StringVal()

	takestamp, _ := x.Get(exif.DateTimeOriginal)
	photo.Takestamp, _ = takestamp.StringVal()

	return nil
}

func (photo *Photo) CopyToUpload() (err error) {
	from, err := os.Open(photo.imagePath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	to, err := os.Create(photo.uploadPath)
	if err != nil {
		log.Error("%v", err)
		return
	}
	_, err = io.Copy(from, to)
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
