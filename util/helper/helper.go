package helper

import (
	"fmt"
	"time"
)

func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
