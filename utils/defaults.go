package utils

import (
	"fmt"
)

func DefaultServerPort() int {
	return 10111
}

func DefaultServerAddress() string {
	return fmt.Sprintf("localhost:%d", DefaultServerPort())
}
