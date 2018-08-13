package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/litao91/lychee_go/modules"
	"github.com/litao91/lychee_go/util/log"
)

func main() {
	wd, err := filepath.Abs(filepath.Dir(os.Args[1]))
	log.Debug("Working directory: %s", wd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	s := modules.NewServer(wd, 3334)
	err = s.Init()
	if err != nil {
		log.Error("%v", err)
	}
	s.Run()
}
