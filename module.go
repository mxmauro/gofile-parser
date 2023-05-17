package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// -----------------------------------------------------------------------------

type Module struct {
	Name   string
	SubDir string
}

// -----------------------------------------------------------------------------

func (m *Module) FullName() string {
	fullName := m.Name
	if len(m.SubDir) > 0 {
		fullName += "/" + m.SubDir
	}
	return fullName
}

func resolveModule(filename string) (Module, error) {
	var f *os.File
	var err error

	// Locate go.mod file in the same or parent subdirectories
	idx := strings.LastIndex(filename, string(os.PathSeparator))
	if idx < 0 {
		return Module{}, fmt.Errorf("unable to locate go.mod file")
	}

	baseDir := filename[0:idx]
	subDirectory := ""
	for {
		f, err = os.Open(baseDir + string(os.PathSeparator) + "go.mod")
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return Module{}, err
		}

		idx = strings.LastIndex(baseDir, string(os.PathSeparator))
		if idx < 0 {
			return Module{}, fmt.Errorf("unable to locate go.mod file")
		}

		dirFragment := baseDir[(idx + 1):]
		if strings.ToLower(dirFragment) == "vendor" {
			// We reached a vendor subdirectory, stop here
			return Module{}, fmt.Errorf("unable to locate go.mod file")
		}
		if len(subDirectory) > 0 {
			subDirectory = dirFragment + "/" + subDirectory
		} else {
			subDirectory = dirFragment
		}

		baseDir = baseDir[0:idx]
	}

	// Found it, begin scan
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	moduleName := ""
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "module ") {
			moduleName = scanner.Text()[7:]
			break
		}
	}
	if scanner.Err() != nil {
		return Module{}, scanner.Err()
	}

	if len(moduleName) == 0 {
		return Module{}, errors.New("go module name not found")
	}

	// Done
	return Module{
		Name:   moduleName,
		SubDir: subDirectory,
	}, nil
}
