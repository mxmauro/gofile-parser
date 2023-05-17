# gofile-parser

This library parses .go files and returns an object containing definitions found on it. It aims 
to be a friendly wrapper around `go/ast`. 

#### Limitations:

* Recognizes interfaces but it doesn't parse its methods.
* Functions and constants are not processed. The initial goal of this library is to return type
  definitions like structs and maps.
* May not work with dot `.` imports.

## Usage

```golang
package main

import (
    parser "github.com/mxmauro/gofile-parser"
)

func main() {
    pf, err := parser.ParseFile(parser.ParseFileOptions{
        Filename:      "./main.go",
        ResolveModule: true,
    })
}
```

`ParseFile`, `ParseDirectory` & `ParseText` parses a go source code from a string or file(s).

`ResolveModule` tries to locate the project's `go.mod` in order to gather the module name it
belongs to and subdirectory.

`ResolveReferences` tries to resolve references, for example, when one struct has a field
pointing to another one.

To process a `ParsedDeclaration`, it is recommended to use `switch v := pd.Type.(type) {`,
where `pd` references to some `ParsedDeclaration`, in order to know the real type of the
declaration.

They can be:

* `ParsedNativeType`
* `ParsedNonNativeType`
* `ParsedStruct`
* `ParsedStructField`
* `ParsedInterface`
* `ParsedMap`
* `ParsedArray`
* `ParsedPointer`
* `ParsedChannel`

The same logic applies when, for example, you want to know the type of object a
`ParsedPointer` points to, or the key and value types of a `ParsedMap`.

## LICENSE

[MIT](/LICENSE)
