package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/litao91/lychee_go/modules"
	"github.com/litao91/lychee_go/util/helper"
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

	pd := path.Join(dd, os.Args[3])
	log.Info("Photo directory " + pd)

	server := modules.NewServer(wd, dd, 3334)
	err = server.Init()
	if err != nil {
		log.Error("%v", err)
	}
	db, err := server.GetDBConnection()
	if err != nil {
		log.Error("%v", err)
		return
	}
	defer db.Close()

	fileList := [][]string{}

	err = filepath.Walk(pd, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(path), ".jpg") {
			fileList = append(fileList, []string{path, f.Name()})
		}
		return nil
	})

	for _, file := range fileList {
		log.Info("Path %s, name %s", file[0], file[1])
		ID := helper.GenerateID()
		p, err := modules.NewPhoto(server, file[0], file[1], ID)
		if err != nil {
			log.Error("%v", err)
			continue
		}
		err = p.SavePhoto(db, false)
		if err != nil {
			log.Error("%v", err)
		}
	}

}
