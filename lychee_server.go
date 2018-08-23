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
	log.Info("Data directory: %s", dd)
	s := modules.NewServer(wd, dd, 3334)
	err = s.Init()
	if err != nil {
		log.Error("%v", err)
		return
	}
	s.Run()
}
