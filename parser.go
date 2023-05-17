package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// -----------------------------------------------------------------------------

type ParsedFile struct {
	Module       Module
	Filename     string
	Package      string
	Imports      []ParsedImports
	Declarations []ParsedDeclaration

	fileContent string
}

type ParsedDeclaration struct {
	Name string
	Type interface{}
	Tags ParsedTags
}

type ParsedNativeType struct {
	Name string
}

type ParsedNonNativeType struct {
	Name string
	Ref  *ParsedDeclaration
}

type ParsedStruct struct {
	Fields []ParsedStructField
}

type ParsedStructField struct {
	Names []string
	Type  interface{}
	Tags  ParsedTags
}

type ParsedInterface struct {
}

type ParsedMap struct {
	KeyType   interface{}
	ValueType interface{}
}

type ParsedArray struct {
	Size      interface{} // Can be empty string, ellipsis, a type
	ValueType interface{}
}

type ParsedPointer struct {
	ToType interface{}
}

type ParsedChannel struct {
	Dir  ast.ChanDir
	Type interface{}
}

type ParsedImports struct {
	Name string
	Path string
}

type ParseTextOptions struct {
	Content  string
	Filename string
	Module   Module
}

// -----------------------------------------------------------------------------

func ParseText(opts ParseTextOptions) (*ParsedFile, error) {
	var fileAst *ast.File
	var err error

	pf := ParsedFile{
		Filename:     opts.Filename,
		Module:       opts.Module,
		Declarations: make([]ParsedDeclaration, 0),
		fileContent:  opts.Content,
	}

	// Parse go file
	fileAst, err = parser.ParseFile(
		token.NewFileSet(), pf.Filename, pf.fileContent,
		parser.SkipObjectResolution|parser.ParseComments,
	)
	if err != nil {
		return nil, err
	}

	// Parse package name
	if fileAst.Name != nil {
		pf.Package = fileAst.Name.Name
	}

	// Parse imports
	for _, imp := range fileAst.Imports {
		pi := ParsedImports{}

		pi.Path, err = strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("unable to parse import %s [err=%v]", imp.Path.Value, err)
		}

		if imp.Name != nil {
			pi.Name = imp.Name.String()
		} else {
			idx := strings.LastIndex(pi.Path, "/")
			if idx >= 0 {
				pi.Name = pi.Path[idx+1:]
			} else {
				pi.Name = pi.Path
			}
		}

		pf.Imports = append(pf.Imports, pi)
	}

	// Parse type declarations
	for _, decl := range fileAst.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok2 := spec.(*ast.TypeSpec); ok2 {
					var decl interface{}

					if len(typeSpec.Name.String()) == 0 {
						continue
					}
					//  || (!typeSpec.Name.IsExported())

					decl, err = pf.convertType(typeSpec.Type)
					if err != nil {
						return nil, fmt.Errorf("unable to parse declaration %s [err=%v]", typeSpec.Name.String(), err)
					}
					if decl == nil {
						continue
					}

					pd := ParsedDeclaration{
						Name: typeSpec.Name.String(),
						Type: decl,
						Tags: make(ParsedTags),
					}

					if genDecl.Doc != nil {
						pd.parseDirectives(genDecl.Doc)
					}
					if typeSpec.Comment != nil {
						pd.parseDirectives(typeSpec.Comment)
					}

					pf.Declarations = append(pf.Declarations, pd)
				}
			}
		}
	}

	// Done
	return &pf, nil
}

func (pf *ParsedFile) convertType(expr ast.Expr) (interface{}, error) {
	var node interface{} = expr

	switch node := node.(type) {
	case *ast.ParenExpr:
		return pf.convertType(node.X)

	case *ast.Ident:
		return pf.parseIdent(node)

	case *ast.StructType:
		return pf.parseStruct(node)

	case *ast.InterfaceType:
		return pf.parseInterface(node)

	case *ast.MapType:
		return pf.parseMap(node)

	case *ast.ArrayType:
		return pf.parseArray(node)

	case *ast.StarExpr:
		return pf.parsePointer(node)

	case *ast.SelectorExpr:
		if xIdent, ok2 := node.X.(*ast.Ident); ok2 {
			return &ParsedNonNativeType{
				Name: xIdent.Name + "." + node.Sel.Name,
			}, nil
		}

	case *ast.ChanType:
		return pf.parseChannel(node)
	}

	// Not supported
	return nil, nil
}

func (pf *ParsedFile) parseIdent(expr ast.Expr) (interface{}, error) {
	ident := expr.(*ast.Ident)

	if IsNativeType(ident.Name) {
		return &ParsedNativeType{
			Name: ident.Name,
		}, nil
	}

	return &ParsedNonNativeType{
		Name: ident.Name,
	}, nil
}

func (pf *ParsedFile) parseStruct(expr ast.Expr) (*ParsedStruct, error) {
	var err error

	st := expr.(*ast.StructType)

	ps := ParsedStruct{}

	if st.Fields != nil && st.Fields.List != nil {
		for _, field := range st.Fields.List {

			psf := ParsedStructField{
				Names: make([]string, 0),
			}

			for _, name := range field.Names {
				psf.Names = append(psf.Names, name.Name)
			}

			if field.Tag != nil {
				psf.Tags = scanTags(field.Tag.Value[1 : len(field.Tag.Value)-1]) // remove side `
			}

			psf.Type, err = pf.convertType(field.Type)
			if err != nil {
				return nil, err
			}

			if psf.Type != nil {
				ps.Fields = append(ps.Fields, psf)
			}
		}
	}

	// Done
	return &ps, nil
}

func (pf *ParsedFile) parseInterface(_ ast.Expr) (*ParsedInterface, error) {
	//var err error

	//it := expr.(*ast.InterfaceType)

	pi := ParsedInterface{}

	// We are not interested in interface methods

	// Done
	return &pi, nil
}

func (pf *ParsedFile) parseMap(expr ast.Expr) (*ParsedMap, error) {
	var err error

	m := expr.(*ast.MapType)

	pm := ParsedMap{}

	pm.KeyType, err = pf.convertType(m.Key)
	if err != nil {
		return nil, err
	}
	pm.ValueType, err = pf.convertType(m.Value)
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (pf *ParsedFile) parseArray(expr ast.Expr) (*ParsedArray, error) {
	var err error

	a := expr.(*ast.ArrayType)

	pa := ParsedArray{}

	pa.ValueType, err = pf.convertType(a.Elt)
	if err != nil {
		return nil, err
	}
	if pa.ValueType == nil {
		return nil, nil
	}
	if a.Len == nil {
		pa.Size = ""
		return &pa, nil
	}

	if _, ok := a.Len.(*ast.Ellipsis); ok {
		pa.Size = "..."
		return &pa, nil
	}

	if lenSelExpr, ok := a.Len.(*ast.SelectorExpr); ok {
		if xIdent, ok2 := lenSelExpr.X.(*ast.Ident); ok2 {
			pa.Size = xIdent.Name + "." + lenSelExpr.Sel.Name
			return &pa, nil
		}
	}

	if lenBasicLit, ok := a.Len.(*ast.BasicLit); ok {
		/*
			switch lenBasicLit.Kind {
			case token.CHAR:
				if strings.HasPrefix(lenBasicLit.Value, "'\\u") || strings.HasPrefix(lenBasicLit.Value, "'\\U") {
					pa.Size = "0x" + lenBasicLit.Value[3:len(lenBasicLit.Value)-1]
					return pa, nil
				}

			default:
				pa.Size = lenBasicLit.Value
				return pa, nil
			}
		*/
		pa.Size = lenBasicLit.Value
		return &pa, nil
	}

	if lenIdent, ok := a.Len.(*ast.Ident); ok {
		pa.Size = lenIdent.Name
		return &pa, nil
	}

	pa.Size = pf.fileContent[a.Len.Pos()-1 : a.Len.End()-1]
	return &pa, nil

	//return ParsedArray{}, errors.New("unsupported size")
}

func (pf *ParsedFile) parsePointer(expr ast.Expr) (*ParsedPointer, error) {
	var err error

	st := expr.(*ast.StarExpr)

	pp := ParsedPointer{}

	pp.ToType, err = pf.convertType(st.X)
	if err != nil {
		return nil, err
	}
	if pp.ToType == nil {
		return nil, errors.New("unsupported pointer to unknown")
	}
	return &pp, nil
}

func (pf *ParsedFile) parseChannel(expr ast.Expr) (*ParsedChannel, error) {
	var err error

	st := expr.(*ast.ChanType)

	pc := ParsedChannel{
		Dir: st.Dir,
	}

	pc.Type, err = pf.convertType(st.Value)
	if err != nil {
		return nil, err
	}
	if pc.Type == nil {
		return nil, errors.New("unsupported channel type")
	}
	return &pc, nil
}

func (pd *ParsedDeclaration) parseDirectives(commentGroup *ast.CommentGroup) {
	for _, line := range strings.Split(commentGroup.Text(), "\n") {
		line := strings.TrimSpace(line)
		for k, v := range scanTags(line) {
			pd.Tags[k] = v
		}
	}
}
