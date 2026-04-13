package log

import (
	"fmt"
	"os"

	"github.com/oisee/vibing-steampunk/internal/config"
)

func Info(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
	}
}

func Warning(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[WARNING] "+format+"\n", args...)
	}
}
