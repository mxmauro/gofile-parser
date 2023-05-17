package parser

import (
	"strconv"
	"strings"
)

// -----------------------------------------------------------------------------

// ParsedTag is a string like the original "reflect" library.
// But this extension allows to split the tag into a comma separated list of
// properties in the format key=value.
type ParsedTag string

type ParsedTags map[string]ParsedTag

// -----------------------------------------------------------------------------

func scanTags(tag string) ParsedTags {
	pts := make(ParsedTags)

	for tag != "" {
		// Skip leading spaces
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := tag[:i]
		tag = tag[i+1:]

		// Scan quoted string to find value.
		for i = 1; i < len(tag) && tag[i] != '"'; i++ {
			if tag[i] == '\\' {
				i++
			}
		}
		if i >= len(tag) {
			break
		}
		quotedValue := tag[:i+1]
		tag = tag[i+1:]

		value, err := strconv.Unquote(quotedValue)
		if err == nil {
			pts[name] = ParsedTag(value)
		}
	}

	return pts
}

func (pts *ParsedTags) HasTag(tag string) bool {
	_, ok := (*pts)[tag]
	return ok
}

func (pts *ParsedTags) GetTag(tag string) (ParsedTag, bool) {
	item, ok := (*pts)[tag]
	return item, ok
}

func (pt ParsedTag) HasProperty(key string) bool {
	_, ok := pt.GetProperty(key)
	return ok
}

func (pt ParsedTag) GetProperty(key string) (string, bool) {
	tagLen := len(pt)

	ofs := 0
	for ofs < tagLen {
		// Skip spaces
		for ofs < tagLen && (pt[ofs] == ' ' || pt[ofs] == '\t') {
			ofs += 1
		}
		if ofs >= tagLen {
			break
		}

		// Scan key name
		keyStartOfs := ofs
		for ofs < tagLen && pt[ofs] != ' ' && pt[ofs] != '\t' && pt[ofs] != '=' && pt[ofs] != ',' {
			ofs += 1
		}
		if keyStartOfs == ofs {
			break // Stop on invalid property list
		}
		isOurKey := key == string(pt[keyStartOfs:ofs])

		// Skip spaces
		for ofs < tagLen && (pt[ofs] == ' ' || pt[ofs] == '\t') {
			ofs += 1
		}

		// No value?
		if ofs >= tagLen || pt[ofs] == ',' {
			if isOurKey {
				return "", true
			}
			if ofs < tagLen {
				ofs += 1
				continue
			}
			break
		}

		// Parse value
		ofs += 1 // Skip equal sign

		// Skip spaces
		for ofs < tagLen && (pt[ofs] == ' ' || pt[ofs] == '\t') {
			ofs += 1
		}
		if ofs >= tagLen {
			if isOurKey {
				return "", true
			}
			break
		}

		// Quoted value?
		value := ""
		if pt[ofs] == '"' || pt[ofs] == '\'' || pt[ofs] == '`' {
			// Advance until closing quote
			ch := pt[ofs]
			ofs += 1
			valueStartOfs := ofs
			for ofs < tagLen && pt[ofs] != ch {
				if pt[ofs] == '\r' && ch == '`' {
					value = value + string(pt[valueStartOfs:ofs])
					ofs += 1
					valueStartOfs = ofs
				} else if pt[ofs] == '\\' && ch != '`' {
					value = value + string(pt[valueStartOfs:ofs])
					ofs += 1
					if ofs >= tagLen {
						break // Closing quotes not found (no need to break+jump because the if after
					}
					valueStartOfs = ofs
					ofs += 1
				} else {
					ofs += 1
				}
			}
			if ofs >= tagLen {
				break // Closing quotes not found
			}
			if valueStartOfs < ofs {
				value = value + string(pt[valueStartOfs:ofs])
			}
			ofs += 1
		} else {
			// Advance until next blank or separator
			valueStartOfs := ofs
			for ofs < tagLen && pt[ofs] > ' ' && pt[ofs] != ',' {
				ofs += 1
			}
			value = string(pt[valueStartOfs:ofs])
		}

		if isOurKey {
			return value, true
		}

		// Skip spaces
		for ofs < tagLen && (pt[ofs] == ' ' || pt[ofs] == '\t') {
			ofs += 1
		}

		// Check for a list separator or end of string
		if ofs < tagLen {
			if pt[ofs] != ',' {
				break
			}
			ofs += 1
		}
	}

	// Not found
	return "", false
}

func (pt ParsedTag) GetBoolProperty(key string) bool {
	value, ok := pt.GetProperty(key)
	if ok {
		if len(value) == 0 || value == "1" {
			return true
		}
		value = strings.ToLower(value)
		if value == "true" || value == "yes" {
			return true
		}
	}
	return false
}
