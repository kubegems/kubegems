package git

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func Test_pulllock_Go(t *testing.T) {
	fun := func() error {
		fmt.Printf("pulllock content running...")
		time.Sleep(time.Second)
		return fmt.Errorf("pulllock content failed: %d", time.Now().UnixNano())
	}

	l := pulllock{}

	runfun := func() {
		if err := l.Go(fun); err != nil {
			log.Printf("pulllock.Go() error = %v", err)
		}
	}

	go runfun()
	go runfun()
	go runfun()
	go runfun()
	go runfun()

	time.Sleep(5 * time.Second)
}
