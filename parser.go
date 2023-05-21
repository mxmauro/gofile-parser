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

type ParsedField struct {
	Names        []string
	ImplicitName string
	Type         interface{}
	Tags         ParsedTags
}

type ParsedStruct struct {
	Fields []ParsedStructField
}

type ParsedStructField = ParsedField

type ParsedInterface struct {
	Methods    []ParsedInterfaceMethod
	Incomplete bool
}

type ParsedInterfaceMethod = ParsedField

type ParsedMap struct {
	KeyType   interface{}
	ValueType interface{}
}

type ParsedArray struct {
	Size      string // Empty string means a slice, ellipsis is an array of items known at compile time, else a constant or a type
	ParsedInt *int64
	ValueType interface{}
}

type ParsedPointer struct {
	ToType interface{}
}

type ParsedChannel struct {
	Dir  ast.ChanDir
	Type interface{}
}

type ParsedFunction struct {
	TypeParams []ParsedFunctionTypeParam
	Params     []ParsedFunctionParam
	Results    []ParsedFunctionResult
}

type ParsedFunctionTypeParam = ParsedField
type ParsedFunctionParam = ParsedField
type ParsedFunctionResult = ParsedField

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

	case *ast.FuncType:
		return pf.parseFunction(node)
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

	ps.Fields, err = pf.parseFields(st.Fields)
	if err != nil {
		return nil, err
	}

	// Done
	return &ps, nil
}

func (pf *ParsedFile) parseInterface(expr ast.Expr) (*ParsedInterface, error) {
	var err error

	it := expr.(*ast.InterfaceType)

	pi := ParsedInterface{
		Incomplete: it.Incomplete,
	}
	pi.Methods, err = pf.parseFields(it.Methods)
	if err != nil {
		return nil, err
	}

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

	// Done
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

		// Done
		return &pa, nil
	}

	if _, ok := a.Len.(*ast.Ellipsis); ok {
		pa.Size = "..."

		// Done
		return &pa, nil
	}

	if lenSelExpr, ok := a.Len.(*ast.SelectorExpr); ok {
		if xIdent, ok2 := lenSelExpr.X.(*ast.Ident); ok2 {
			pa.Size = xIdent.Name + "." + lenSelExpr.Sel.Name

			// Done
			return &pa, nil
		}
	}

	if lenBasicLit, ok := a.Len.(*ast.BasicLit); ok {
		switch lenBasicLit.Kind {
		case token.INT:
			if parsedInt, err2 := strconv.ParseInt(lenBasicLit.Value, 0, 64); err2 == nil {
				pa.ParsedInt = &parsedInt
			}

		case token.CHAR:
			if n := len(lenBasicLit.Value); n >= 2 {
				if code, _, _, err2 := strconv.UnquoteChar(lenBasicLit.Value[1:n-1], '\''); err2 == nil {
					parsedInt := int64(code)
					pa.ParsedInt = &parsedInt
				}
			}
		}

		pa.Size = lenBasicLit.Value

		// Done
		return &pa, nil
	}

	if lenIdent, ok := a.Len.(*ast.Ident); ok {
		pa.Size = lenIdent.Name

		// Done
		return &pa, nil
	}

	pa.Size = pf.fileContent[a.Len.Pos()-1 : a.Len.End()-1]

	// Done
	return &pa, nil
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

	// Done
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

	// Done
	return &pc, nil
}

func (pf *ParsedFile) parseFunction(expr ast.Expr) (*ParsedFunction, error) {
	var err error

	ft := expr.(*ast.FuncType)

	pfunc := ParsedFunction{}

	pfunc.TypeParams, err = pf.parseFields(ft.TypeParams)
	if err != nil {
		return nil, err
	}
	pfunc.Params, err = pf.parseFields(ft.Params)
	if err != nil {
		return nil, err
	}
	pfunc.Results, err = pf.parseFields(ft.Results)
	if err != nil {
		return nil, err
	}

	// Done
	return &pfunc, nil
}

func (pf *ParsedFile) parseFields(fields *ast.FieldList) ([]ParsedField, error) {
	var err error

	fieldsList := make([]ParsedField, 0)

	if fields != nil && fields.List != nil {
		for _, field := range fields.List {
			pfld := ParsedField{
				Names: make([]string, 0),
			}

			for _, name := range field.Names {
				pfld.Names = append(pfld.Names, name.Name)
			}

			if field.Tag != nil {
				pfld.Tags = scanTags(field.Tag.Value[1 : len(field.Tag.Value)-1]) // remove side `
			}

			pfld.Type, err = pf.convertType(field.Type)
			if err != nil {
				return nil, err
			}

			if pfld.Type != nil {
				if len(pfld.Names) == 0 {
					pfld.ImplicitName = guessImplicitName(pfld.Type)
				}

				fieldsList = append(fieldsList, pfld)
			}
		}
	}

	// Done
	return fieldsList, err
}

func (pd *ParsedDeclaration) parseDirectives(commentGroup *ast.CommentGroup) {
	for _, line := range strings.Split(commentGroup.Text(), "\n") {
		line := strings.TrimSpace(line)
		for k, v := range scanTags(line) {
			pd.Tags[k] = v
		}
	}
}

// -----------------------------------------------------------------------------

func guessImplicitName(t interface{}) string {
	switch tType := t.(type) {
	case *ParsedNativeType:
		return tType.Name

	case *ParsedNonNativeType:
		_, name := GetIdentifierParts(tType.Name)
		return name

	case *ParsedArray:
		return guessImplicitName(tType.ValueType)

	case *ParsedPointer:
		return guessImplicitName(tType.ToType)

	case *ParsedChannel:
		return guessImplicitName(tType.Type)
	}
	return ""
}
