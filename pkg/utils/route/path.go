package route

type PathTokens struct {
	Tokens []string
}

func ParsePathTokens(path string) []string {
	tokens := []string{}
	pos := 0
	for i, char := range path {
		if char == '/' {
			if pos != i {
				tokens = append(tokens, path[pos:i])
			}
			tokens = append(tokens, "/")
			pos = i + 1
		}
	}
	if pos != len(path) {
		tokens = append(tokens, path[pos:])
	}
	return tokens
}

func CompilePathPattern(pattern string) ([][]Element, error) {
	sections := [][]Element{}
	pathtokens := ParsePathTokens(pattern)
	for _, token := range pathtokens {
		if token == "/" {
			sections = append(sections, []Element{{kind: ElementKindSplit}})
			continue
		}
		compiled, err := CompileSection(token)
		if err != nil {
			return nil, err
		}
		sections = append(sections, compiled)
	}
	return sections, nil
}
