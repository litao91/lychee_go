package sqlite_sql

import (
	"database/sql"

	"github.com/litao91/lychee_go/util/log"
	_ "github.com/mattn/go-sqlite3"
)

var CreateTableStmt string = `
CREATE TABLE IF NOT EXISTS lychee_albums (
  id bigint(14) NOT NULL,
  title varchar(100) NOT NULL DEFAULT '',
  description varchar(1000) DEFAULT '',
  sysstamp int(11) NOT NULL,
  public tinyint(1) NOT NULL DEFAULT '0',
  visible tinyint(1) NOT NULL DEFAULT '1',
  downloadable tinyint(1) NOT NULL DEFAULT '0',
  password varchar(100) DEFAULT NULL,
  PRIMARY KEY (id)
);


CREATE TABLE IF NOT EXISTS lychee_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  time int(11) NOT NULL,
  type varchar(11) NOT NULL,
  function varchar(100) NOT NULL,
  line int(11) NOT NULL,
  text text
);


CREATE TABLE IF NOT EXISTS lychee_photos (
  id bigint(14) NOT NULL,
  title varchar(100) NOT NULL DEFAULT '',
  description varchar(1000) DEFAULT '',
  url varchar(100) NOT NULL,
  tags varchar(1000) NOT NULL DEFAULT '',
  public tinyint(1) NOT NULL,
  type varchar(10) NOT NULL,
  width int(11) NOT NULL,
  height int(11) NOT NULL,
  size varchar(20) NOT NULL,
  iso varchar(15) NOT NULL,
  aperture varchar(20) NOT NULL,
  make varchar(50) NOT NULL,
  model varchar(50) NOT NULL,
  shutter varchar(30) NOT NULL,
  focal varchar(20) NOT NULL,
  takestamp int(11) DEFAULT NULL,
  star tinyint(1) NOT NULL,
  thumbUrl char(37) NOT NULL,
  album bigint(20) NOT NULL,
  checksum char(40) DEFAULT NULL,
  medium varchar(100) NOT NULL DEFAULT '',
  PRIMARY KEY (id)
);


CREATE TABLE IF NOT EXISTS lychee_settings (
  key varchar(50) NOT NULL DEFAULT '',
  value varchar(200) DEFAULT ''
);

INSERT INTO lychee_settings (key, value)
VALUES
  ('version',''),
  ('username',''),
  ('password',''),
  ('checkForUpdates','1'),
  ('sortingPhotos','ORDER BY id DESC'),
  ('sortingAlbums','ORDER BY id DESC'),
  ('imagick','1'),
  ('dropboxKey',''),
  ('identifier',''),
  ('skipDuplicates','0'),
  ('plugins','');
`

func InitTables(dbpath string) (err error) {
	db, err := sql.Open("sqlite3", dbpath)
	defer db.Close()
	if err != nil {
		log.Error("Can't open db on %s: %v", dbpath, err)
		return
	}
	_, err = db.Exec(CreateTableStmt)
	return
}
