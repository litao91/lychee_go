package modules

import (
	"database/sql"

	"github.com/litao91/lychee_go/sqlite_sql"
	_ "github.com/mattn/go-sqlite3"
)

func NewLycheeDb(path string) *LycheeDb {
	return &LycheeDb{
		dbPath: path,
	}

}

type LycheeDb struct {
	dbPath string
}

func (db *LycheeDb) InitDb() (err error) {
	err = sqlite_sql.InitTables(db.dbPath)
	return
}

func (db *LycheeDb) GetConnection() (conn *sql.DB, err error) {
	conn, err = sql.Open("sqlite3", db.dbPath)
	return
}
