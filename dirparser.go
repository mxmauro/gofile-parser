package parser

import (
	"os"
	"strings"
)

// -----------------------------------------------------------------------------

type ParseDirectoryOptions struct {
	BaseDir          string
	ResolveModule    bool
	IncludeTestFiles bool
}

type dirParser struct {
	BaseDir          string
	ResolveModule    bool
	IncludeTestFiles bool
	ParsedFiles      []*ParsedFile
}

// -----------------------------------------------------------------------------

func ParseDirectory(opts ParseDirectoryOptions) ([]*ParsedFile, error) {
	dp := dirParser{
		BaseDir:          opts.BaseDir,
		ResolveModule:    opts.ResolveModule,
		IncludeTestFiles: opts.IncludeTestFiles,
		ParsedFiles:      make([]*ParsedFile, 0),
	}
	if !strings.HasSuffix(dp.BaseDir, string(os.PathSeparator)) {
		dp.BaseDir += string(os.PathSeparator)
	}
	err := dp.parseRecursive("")
	if err != nil {
		return nil, err
	}
	return dp.ParsedFiles, nil
}

func (dp *dirParser) parseRecursive(subDir string) error {
	// Skip vendor and testdata subdirectories
	if subDirEndsWith(subDir, "vendor") || subDirEndsWith(subDir, "testdata") {
		return nil
	}

	files, err := os.ReadDir(dp.BaseDir + subDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			if strings.HasSuffix(file.Name(), ".go") {
				if dp.IncludeTestFiles || (!strings.HasSuffix(file.Name(), "_test.go")) {
					var pf *ParsedFile

					pf, err = ParseFile(ParseFileOptions{
						Filename:      dp.BaseDir + subDir + file.Name(),
						ResolveModule: dp.ResolveModule,
					})
					if err != nil {
						return err
					}
					dp.ParsedFiles = append(dp.ParsedFiles, pf)
				}
			}
		} else if file.Name() != "." && file.Name() != ".." {
			err = dp.parseRecursive(subDir + file.Name() + string(os.PathSeparator))
			if err != nil {
				return err
			}
		}
	}

	// Done
	return nil
}
