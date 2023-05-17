package parser

import (
	"os"
	"strings"
)

// -----------------------------------------------------------------------------

func subDirEndsWith(subDir, fragment string) bool {
	subDir = strings.ToLower(subDir)
	fragment += string(os.PathSeparator)
	return subDir == fragment || strings.HasSuffix(subDir, string(os.PathSeparator)+fragment)
}
