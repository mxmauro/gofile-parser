package parser

import (
	"strings"
)

// -----------------------------------------------------------------------------

func IsPublic(name string) bool {
	if len(name) == 0 || name[0] == '_' {
		return false
	}
	s := string(name[0])
	firstLetter := strings.ToLower(s)
	return firstLetter != s
}

func IsNativeType(name string) bool {
	switch name {
	case "bool", "byte", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "string", "float", "float64", "float32", "complex128", "complex64", "uintptr":
		return true
	}
	return false
}

func GetIdentifierParts(qualifiedIdentifier string) (packageName string, identifier string) {
	dotIdx := strings.Index(qualifiedIdentifier, ".")
	if dotIdx >= 0 {
		packageName = qualifiedIdentifier[0:dotIdx]
		identifier = qualifiedIdentifier[(dotIdx + 1):]
	} else {
		packageName = ""
		identifier = qualifiedIdentifier
	}
	return
}
