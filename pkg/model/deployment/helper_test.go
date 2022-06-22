package deployment

import (
	"fmt"
	"testing"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestRandStringRunes(t *testing.T) {
	randlen := 10
	for i := 0; i < 20; i++ {
		got := RandStringRunes(randlen)
		fmt.Printf("RandStringRunes() = %v\n", got)
	}
}
