package main

import (
	"fmt"

	"kubegems.io/pkg/utils"
)

type A struct {
	n int
	m map[string]string
}

func main() {
	fmt.Printf("appflow-%s", string(utils.RandomRune(4, utils.RuneKindLower)))
}
