package main

import (
	"os"
	"path/filepath"
)

type LchyeeServe struct {
	host     string
	port     int64
	basePath string
}

func main() {
	wd, err := filepath.Abs(filepath.Dir(os.Args[0]))
}
