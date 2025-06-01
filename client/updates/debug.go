package updates

import (
	"log"
	"os"
	"strings"
)

var isDebugEnabled bool

func init() {
	// Check DEBUG environment variable
	debugEnv := strings.ToLower(os.Getenv("DEBUG"))
	isDebugEnabled = debugEnv == "1" || debugEnv == "true"
}

// debugLog prints a message only if debug logging is enabled
func debugLog(format string, v ...interface{}) {
	if isDebugEnabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}
