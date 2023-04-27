package auth

import (
	"strings"
)

const (
	sectionSep  = ":"  // section separator
	multiSep    = ","  // match any of these sections
	wildcard    = "*"  // match this section
	wildcardAll = "**" // match this section and all following sections
)

// acting like: https://shiro.apache.org/permissions.html#WildcardPermissions
// but extended to support ** to match all following sections
func WildcardMatchSections(expr string, perm string) bool {
	exprs, perms := parseSections(expr), parseSections(perm)
	exprssize, permssize := len(exprs), len(perms)
	for i, permsec := range perms {
		if i >= exprssize {
			return false // perm has more sections than expr
		}
		exprsec := exprs[i]
		if contains(exprsec, wildcardAll) {
			return true // expr has wildcardAll, so it matches all remaining sections
		}
		if !contains(exprsec, wildcard) && !containsAll(exprsec, permsec) {
			return false // expr has no wildcard, so permsec must be in exprsec
		}
	}
	for i := permssize; i < exprssize; i++ {
		if contains(exprs[i], wildcardAll) {
			return true // perm has fewer sections than expr, but expr has wildcardAll, so it matches all remaining sections
		}
		if !contains(exprs[i], wildcard) {
			return false // perm has fewer sections than expr, but expr has no wildcard, so it must not match
		}
	}
	return true
}

func parseSections(perm string) [][]string {
	perm = strings.TrimSpace(perm)
	if perm == "" {
		return nil
	}
	sectionstrs := strings.Split(perm, sectionSep)
	sections := make([][]string, len(sectionstrs))
	for i, section := range sectionstrs {
		sections[i] = strings.Split(section, multiSep)
	}
	return sections
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

func containsAll(arr []string, s []string) bool {
	for _, v := range s {
		if !contains(arr, v) {
			return false
		}
	}
	return true
}

func removeEmpty(arr []string) []string {
	w := 0
	for _, v := range arr {
		if v != "" {
			arr[w] = v
			w++
		}
	}
	return arr[:w]
}
