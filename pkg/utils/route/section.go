package route

import (
	"fmt"
	"strings"
)

type ElementKind string

const (
	ElementKindNone     ElementKind = ""
	ElementKindConst    ElementKind = "const"
	ElementKindVariable ElementKind = "{}"
	ElementKindStar     ElementKind = "*"
	ElementKindSplit    ElementKind = "/"
)

type Element struct {
	kind  ElementKind
	param string
}

type CompileError struct {
	Pattern  string
	Position int
	Rune     rune
	Message  string
}

func (e CompileError) Error() string {
	return fmt.Sprintf("invalid char [%c] in [%s] at position %d: %s", e.Rune, e.Pattern, e.Position, e.Message)
}

func MustCompileSection(pattern string) []Element {
	ret, err := CompileSection(pattern)
	if err != nil {
		panic(err)
	}
	return ret
}

func CompileSection(pattern string) ([]Element, error) {
	elems := []Element{}

	patternlen := len(pattern)
	hasStarSuffix := false
	if pattern[patternlen-1] == '*' {
		pattern = pattern[:patternlen-1]
		hasStarSuffix = true
	}

	pos := 0
	currentKind := ElementKindNone
	for i, rune := range pattern {
		switch {
		case rune == '{' && currentKind != ElementKindVariable:
			// end a const definition
			if currentKind == ElementKindConst {
				elems = append(elems, Element{kind: ElementKindConst, param: pattern[pos:i]})
			}
			// start a variable defination
			currentKind = ElementKindVariable
			pos = i + 1

		case rune == '}' && currentKind == ElementKindVariable:
			// end a variable defination
			elems = append(elems, Element{kind: ElementKindVariable, param: pattern[pos:i]})
			currentKind = ElementKindNone
			pos = i + 1
		default:
			// if in a variable difinarion or a const define
			if currentKind == ElementKindVariable || currentKind == ElementKindConst {
				continue
			}
			// start a const defination if not
			if currentKind != ElementKindConst {
				currentKind = ElementKindConst
				pos = i
			}
		}
	}
	// last
	switch currentKind {
	case ElementKindConst:
		// end a const definition
		elems = append(elems, Element{kind: ElementKindConst, param: pattern[pos:]})
	case ElementKindVariable:
		return nil, CompileError{Position: len(pattern), Pattern: pattern, Rune: rune(pattern[len(pattern)-1]), Message: "variable defination not closed"}
	}

	if hasStarSuffix {
		elems = append(elems, Element{kind: ElementKindStar})
	}
	return elems, nil
}

func MatchSection(compiled []Element, sections []string) (bool, bool, map[string]string) {
	vars := map[string]string{}
	if len(sections) == 0 {
		return false, false, nil
	}

	section := sections[0]

	pos := 0
	for i, elem := range compiled {
		switch elem.kind {
		case ElementKindConst:
			conslen := len(elem.param)
			if len(section) < pos+conslen {
				return false, false, nil
			}
			str := section[pos : pos+conslen]
			if str != elem.param {
				return false, false, nil
			}
			// next section mactch
			pos += conslen
		case ElementKindVariable:
			// no next
			if i == len(compiled)-1 {
				vars[elem.param] = section[pos:]
				return true, false, vars
			}
			// if next is const
			nextsec := compiled[i+1]

			switch nextsec.kind {
			case ElementKindConst:
				index := strings.Index(section[pos:], nextsec.param)
				if index == -1 || index == 0 {
					// not match next const
					return false, false, nil
				}
				// var is bettwen pos and next sec start
				vars[elem.param] = section[pos : pos+index]
				pos += index
			case ElementKindVariable:
				continue
			case ElementKindStar:
				// var is bettwen pos to sections end
				vars[elem.param] = strings.Join(append([]string{section[pos:]}, sections[1:]...), "")
				return true, true, vars
			}
		case ElementKindStar:
			return true, true, vars
		case ElementKindSplit:
			if section == "/" {
				return true, false, vars
			}
			return false, false, nil
		}
	}
	// section left some chars
	// eg.  {kind:const,param:api} and apis; 's' remain
	if section[pos:] != "" {
		return false, false, nil
	}
	return true, false, vars
}
