package parser

import (
	"os"
	"path/filepath"
)

// -----------------------------------------------------------------------------

type ParseFileOptions struct {
	Filename      string
	ResolveModule bool
}

// -----------------------------------------------------------------------------

func ParseFile(opts ParseFileOptions) (*ParsedFile, error) {
	var fileContent []byte
	var module Module

	filename, err := filepath.Abs(opts.Filename)
	if err != nil {
		return nil, err
	}

	// Read go file
	fileContent, err = os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Resolve module name if requested
	if opts.ResolveModule {
		module, err = resolveModule(filename)
		if err != nil {
			return nil, err
		}
	}

	// Parse context
	return ParseText(ParseTextOptions{
		Content:  string(fileContent),
		Filename: filename,
		Module:   module,
	})
}
