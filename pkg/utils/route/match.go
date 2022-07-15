// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package route

import (
	"fmt"
	"sort"
)

type matcher struct {
	root *node
}

type node struct {
	key      []Element
	val      *matchitem
	children []*node
}

func (n *node) indexChild(s *node) int {
	for index, child := range n.children {
		if isSamePattern(child.key, s.key) {
			return index
		}
	}
	return -1
}

func isSamePattern(a, b []Element) bool {
	tostr := func(elems []Element) string {
		str := ""
		for _, e := range elems {
			switch e.kind {
			case ElementKindConst:
				str += e.param
			case ElementKindVariable:
				str += "{}"
			case ElementKindStar:
				str += "*"
			case ElementKindSplit:
				str += "/"
			}
		}
		return str
	}
	return tostr(a) == tostr(b)
}

func sortSectionMatches(sections []*node) {
	sort.Slice(sections, func(i, j int) bool {
		secsi, secsj := sections[i].key, sections[j].key

		switch lasti, lastj := (secsi)[len(secsi)-1].kind, (secsj)[len(secsj)-1].kind; {
		case lasti == ElementKindStar && lastj != ElementKindStar:
			return false
		case lasti != ElementKindStar && lastj == ElementKindStar:
			return true
		}
		cnti, cntj := 0, 0
		for _, v := range secsi {
			switch v.kind {
			case ElementKindConst:
				cnti += 99
			case ElementKindVariable:
				cnti -= 1
			}
		}

		for _, v := range secsj {
			switch v.kind {
			case ElementKindConst:
				cntj += 99
			case ElementKindVariable:
				cntj -= 1
			}
		}

		return cnti > cntj
	})
}

type matchitem struct {
	pattern string
	val     interface{} // of val not nil. it's the matched
}

func (m *matcher) Register(pattern string, val interface{}) error {
	sections, err := CompilePathPattern(pattern)
	if err != nil {
		return err
	}
	item := &matchitem{pattern: pattern, val: val}

	cur := m.root
	for i, section := range sections {
		child := &node{key: section}
		if index := cur.indexChild(child); index == -1 {
			if i == len(sections)-1 {
				child.val = item
			}
			cur.children = append(cur.children, child)
			sortSectionMatches(cur.children)
		} else {
			child = cur.children[index]
			if i == len(sections)-1 {
				if child.val != nil {
					return fmt.Errorf("pattern %s conflicts with exists %s", pattern, child.val.pattern)
				}
				child.val = item
			}
		}
		cur = child
	}
	return nil
}

func (m *matcher) Match(path string) (bool, interface{}, map[string]string) {
	pathtokens := ParsePathTokens(path)

	vars := map[string]string{}
	match := matchchildren(m.root, pathtokens, vars)
	if match == nil {
		return false, nil, vars
	}
	return true, match.val, vars
}

func matchchildren(cur *node, tokens []string, vars map[string]string) *matchitem {
	if len(tokens) == 0 {
		return nil
	}
	var matched *matchitem
	for _, child := range cur.children {
		if matched, matchlefttokens, secvars := MatchSection(child.key, tokens); matched {
			if child.val != nil && len(tokens) == 1 || matchlefttokens {
				mergeMap(secvars, vars)
				return child.val
			}
			result := matchchildren(child, tokens[1:], secvars)
			if result != nil {
				mergeMap(secvars, vars)
				return result
			}
		}
	}
	return matched
}

func mergeMap(src, dst map[string]string) map[string]string {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
