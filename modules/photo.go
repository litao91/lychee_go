package modules

import (
	"database/sql"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/litao91/lychee_go/util/helper"
	"github.com/litao91/lychee_go/util/log"
	"github.com/nfnt/resize"
)

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
	ThumbUrl    string `json:"thumburl"`
	Album       int    `json:"album"`
	Checksum    string `json:"checksum"`
	Medium      int    `json:"medium"`
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
		r, err = loadRow(rows)
		if err != nil {
			log.Error("%v", err)
			return
		}
		photos = append(photos, r)
	}
	return
}

func loadRow(row *sql.Rows) (r *Photo, err error) {
	r = &Photo{}
	err = row.Scan(&r.ID, &r.Title, &r.Description, &r.Url, &r.Tags, &r.Public, &r.Type, &r.Width, &r.Height,
		&r.Size, &r.Iso, &r.Aperture, &r.Make, &r.Model, &r.Shutter, &r.Focal, &r.Takestamp, &r.Star,
		&r.ThumbUrl, &r.Album, &r.Checksum, &r.Medium)
	return
}

func validateExtension(filename string) (string, bool) {
	isValid := strings.HasSuffix(strings.ToLower(filename), ".jpg")
	return "jpg", isValid
}

func prepareDataDirs(dataPath string) (tmpdir, uploadsdir, thumbsdir, mediumdir string) {
	tmpdir = path.Join(dataPath, "tmp")
	if _, err := os.Stat(tmpdir); os.IsNotExist(err) {
		os.Mkdir(tmpdir, 0755)
	}
	uploadsdir = path.Join(dataPath, "uploads")
	if _, err := os.Stat(uploadsdir); os.IsNotExist(err) {
		os.Mkdir(uploadsdir, 0755)
	}
	thumbsdir = path.Join(dataPath, "thumbs")
	if _, err := os.Stat(thumbsdir); os.IsNotExist(err) {
		os.Mkdir(thumbsdir, 0755)
	}
	mediumdir = path.Join(dataPath, "medium")
	if _, err := os.Stat(thumbsdir); os.IsNotExist(err) {
		os.Mkdir(mediumdir, 0755)
	}
	return
}

func UploadAction(server *LycheeServer, c *gin.Context) {
	albumId, err := strconv.Atoi(c.PostForm("albumID"))
	log.Debug("Uploading image to album: %d", albumId)
	if err != nil {
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

	tmpdir, uploadsdir, thumbsdir, mediumdir := prepareDataDirs(server.dataPath)

	tmpFilepath := path.Join(tmpdir, id)
	if err := c.SaveUploadedFile(file, tmpFilepath); err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}
	//TODO verify photo type

	filename := id + "_" + file.Filename
	checksum, err := helper.HashFileSha1(tmpFilepath)
	if err != nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	if doesPhotoExists(checksum) {
		c.JSON(http.StatusBadRequest, "Photo exists")
		return
	}
	pathRelativeToData := "uploads/" + filename
	destpath := path.Join(uploadsdir, pathRelativeToData)
	err = helper.CopyFile(tmpFilepath, destpath)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}

	conn, err := server.db.GetConnection()
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}
	defer conn.Close()

	err = SavePhotoMeta(server.dataPath, thumbsdir, mediumdir, pathRelativeToData, conn, checksum)
	if err != nil {
		log.Error("%v", err)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}

	c.JSON(200, fmt.Sprintf("%d", albumId))
}

func toUrl(relativePath string) string {
	return "data/" + relativePath
}

func SavePhotoMeta(dataDir string, thumbsdir string, mediumdir string, relativePath string, conn *sql.DB, checksum string) error {
	imgPath := path.Join(dataDir, relativePath)
	file, err := os.Open(imgPath)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	mediumRelative := path.Join(mediumdir, checksum+".jpg")
	thumbRelative := path.Join(thumbsdir, checksum+".jpg")
	hasMediumCreated := createMedium(img, path.Join(mediumdir, mediumRelative))
	medium := 0
	if hasMediumCreated {
		medium = 1
	}
	return nil
}

func createThumb(img image.Image, destPath string) error {
	return nil
}

func createMedium(img image.Image, destPath string) bool {
	height := img.Bounds().Size().Y
	width := img.Bounds().Size().X
	if height <= 1920 && width <= 1920 {
		return false
	}
	var newWidth uint = 1920
	if width < height {
		newWidth = 1080
	}
	m := resize.Resize(newWidth, 0, img, resize.Lanczos3)
	out, err := os.Create(destPath)
	if err != nil {
		log.Error("%v", err)
		return false
	}
	defer out.Close()
	jpeg.Encode(out, m, nil)
	return true
}

func doesPhotoExists(checksum string) bool {
	// Place holder for now
	return false
}
