package parser_test

import (
	"testing"

	parser "github.com/mxmauro/gofile-parser"
)

//------------------------------------------------------------------------------

func TestParser(t *testing.T) {
	pfs := make([]*parser.ParsedFile, 0)

	pf, err := parser.ParseText(parser.ParseTextOptions{
		Content: `
package main

import (
	"fmt"
)

type A struct {
	I int ` + "`parser-test-tag:\"this is a test\"`" + `
}

// parser-test-tag:"this is a test"
type B struct {
	A
	J string
}

type C struct {
	B B
	K []byte
	L [10]string
}

func main() {
	c := C{}
	fmt.Println("Hello!", c.B.I)
}
`,
		Filename: "test.go",
		Module: parser.Module{
			Name:   "github.com/mxmauro/gofile-parser-test",
			SubDir: "",
		},
	})
	if err != nil {
		t.Fatalf("%v", err.Error())
	}
	pfs = append(pfs, pf)

	parser.ResolveReferences(pfs)

	v, _ := pfs[0].Declarations[0].Type.(*parser.ParsedStruct).Fields[0].Tags.GetTag("parser-test-tag")
	if v != "this is a test" {
		t.Fatalf("wrong tag")
	}

	v, _ = pfs[0].Declarations[1].Tags.GetTag("parser-test-tag")
	if v != "this is a test" {
		t.Fatalf("wrong tag")
	}

	if pfs[0].Declarations[1].Type.(*parser.ParsedStruct).Fields[0].Type.(*parser.ParsedNonNativeType).Ref != &pfs[0].Declarations[0] {
		t.Fatalf("wrong reference")
	}

	if pfs[0].Declarations[2].Type.(*parser.ParsedStruct).Fields[0].Type.(*parser.ParsedNonNativeType).Ref != &pfs[0].Declarations[1] {
		t.Fatalf("wrong reference")
	}

	if pfs[0].Declarations[2].Type.(*parser.ParsedStruct).Fields[2].Type.(*parser.ParsedArray).ParsedInt == nil {
		t.Fatalf("unable to indentify array size")
	}
}
