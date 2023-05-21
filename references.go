package parser

import (
	"strings"
)

// -----------------------------------------------------------------------------

type refResolver struct {
	ModulesMap map[string][]*ParsedFile

	currentFile *ParsedFile
	currentDecl *ParsedDeclaration
}

// -----------------------------------------------------------------------------

// ResolveReferences tries to resolve all ParsedNonNativeType references
func ResolveReferences(parsedFiles []*ParsedFile) {
	rr := refResolver{
		ModulesMap: make(map[string][]*ParsedFile),
	}

	// Create a map of files classified by module and package
	for _, pf := range parsedFiles {
		moduleName := pf.Module.FullName()

		pFiles, ok := rr.ModulesMap[moduleName]
		if !ok {
			pFiles = make([]*ParsedFile, 0)
		}
		rr.ModulesMap[moduleName] = append(pFiles, pf)
	}

	for _, pf := range parsedFiles {
		rr.currentFile = pf
		for pdIdx, pd := range pf.Declarations {
			rr.currentDecl = &pf.Declarations[pdIdx]
			rr.resolve(pd.Type)
		}
	}
}

func (rr *refResolver) resolve(t interface{}) {
	switch tType := t.(type) {
	case *ParsedNonNativeType:
		rr.resolveNonNativeType(tType)
	case *ParsedStruct:
		rr.processStruct(tType)
	case *ParsedInterface:
		rr.processInterface(tType)
	case *ParsedArray:
		rr.processArray(tType)
	case *ParsedMap:
		rr.processMap(tType)
	case *ParsedPointer:
		rr.processPointer(tType)
	case *ParsedChannel:
		rr.processChannel(tType)
	case *ParsedFunction:
		rr.processFunction(tType)
	}
}

func (rr *refResolver) processStruct(ps *ParsedStruct) {
	for _, field := range ps.Fields {
		rr.resolve(field.Type)
	}
}

func (rr *refResolver) processInterface(pi *ParsedInterface) {
	for _, field := range pi.Methods {
		rr.resolve(field.Type)
	}
}

func (rr *refResolver) processArray(pa *ParsedArray) {
	rr.resolve(pa.ValueType)
}

func (rr *refResolver) processMap(pm *ParsedMap) {
	rr.resolve(pm.KeyType)
	rr.resolve(pm.ValueType)
}

func (rr *refResolver) processPointer(pp *ParsedPointer) {
	rr.resolve(pp.ToType)
}

func (rr *refResolver) processChannel(pc *ParsedChannel) {
	rr.resolve(pc.Type)
}

func (rr *refResolver) processFunction(pf *ParsedFunction) {
	for _, field := range pf.TypeParams {
		rr.resolve(field.Type)
	}
	for _, field := range pf.Params {
		rr.resolve(field.Type)
	}
	for _, field := range pf.Results {
		rr.resolve(field.Type)
	}
}

func (rr *refResolver) resolveNonNativeType(pnnt *ParsedNonNativeType) {
	if pnnt.Ref != nil {
		return // Already resolved
	}

	moduleName := rr.currentFile.Module.FullName()

	// Assume the local reference by default
	objName := pnnt.Name

	dotIdx := strings.Index(pnnt.Name, ".")
	if dotIdx >= 0 {
		// Find the import
		importedPackageName := pnnt.Name[:dotIdx]
		objName = objName[(dotIdx + 1):]

		importPath := ""
		for _, pi := range rr.currentFile.Imports {
			if pi.Name == importedPackageName {
				importPath = pi.Path
				break
			}
		}
		if len(importPath) == 0 {
			return // Unable to determine import path, let's continue
		}

		if importPath != "." {
			if strings.HasPrefix(importPath, ".") {
				// Assume a relative path
				tempPath := importPath
				if len(rr.currentFile.Module.SubDir) > 0 {
					tempPath = rr.currentFile.Module.SubDir + "/" + tempPath
				}

				fragments := strings.Split(tempPath, "/")
				idx := 0
				for idx < len(fragments) {
					if fragments[idx] == "." {
						fragments = append(fragments[0:idx], fragments[(idx+1):]...)
					} else if fragments[idx] == ".." {
						if idx == 0 {
							return // Invalid path
						}
						fragments = append(fragments[0:(idx-1)], fragments[(idx+1):]...)
						idx -= 1
					} else {
						idx += 1
					}
				}

				moduleName = rr.currentFile.Module.Name
				if len(fragments) > 0 {
					moduleName += "/" + strings.Join(fragments, "/")
				}

			} else {
				moduleName = importPath
			}
		}
	}

	parsedFilesToCheck, ok := rr.ModulesMap[moduleName]
	if (!ok) || len(parsedFilesToCheck) == 0 {
		return // Unable to find other modules
	}

	for _, pf2c := range parsedFilesToCheck {
		for pd2cIdx := range pf2c.Declarations {
			pd2c := &pf2c.Declarations[pd2cIdx]
			if pd2c.Name == objName {
				// Found the reference
				pnnt.Ref = pd2c
				return
			}
		}
	}
}
