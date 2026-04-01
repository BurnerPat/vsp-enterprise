package log

import (
	"fmt"
	"os"

	"github.com/oisee/vibing-steampunk/internal/config"
)

func LogInfo(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
	}
}

func LogWarning(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[WARNING] "+format+"\n", args...)
	}
}

func LogError(format string, args ...interface{}) {
	if config.GetInstance().Verbose {
		fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
	}
}

func ShowError(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
