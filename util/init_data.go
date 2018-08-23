package main

import (
	"os"
	"path/filepath"

	"github.com/litao91/lychee_go/modules"
	"github.com/litao91/lychee_go/util/log"
)

func main() {
	wd, err := filepath.Abs(filepath.Dir(os.Args[1]))
	log.Info("Working directory: %s", wd)
	if err != nil {
		log.Error("%v", err)
	}
	dd, err := filepath.Abs(filepath.Dir(os.Args[2]))
	if err != nil {
		log.Error("%v", err)
		return
	}
	log.Info("Data directory: " + dd)
	pd, err := filepath.Abs(filepath.Dir(os.Args[2]))
	if err != nil {
		log.Error("%v", err)
	}
	log.Info("Photo directory " + pd)

	server := modules.NewServer(wd, dd, 3334)
	db, err := server.GetDBConnection()
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer db.Close()
}
