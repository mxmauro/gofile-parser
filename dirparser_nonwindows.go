//go:build !windows
// +build !windows

package parser

import (
	"os"
	"strings"
)

// -----------------------------------------------------------------------------

func subDirEndsWith(subDir, fragment string) bool {
	fragment += string(os.PathSeparator)
	return subDir == fragment || strings.HasSuffix(subDir, string(os.PathSeparator)+fragment)
}
