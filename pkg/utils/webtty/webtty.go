package webtty

import (
	"context"
	"os/exec"

	"k8s.io/client-go/tools/remotecommand"
)

func Exec(ctx context.Context, cmd string, args []string, options remotecommand.StreamOptions) error {
	execCmd := exec.CommandContext(ctx, cmd, args...)
	if options.Tty {
		return TTYCmd(execCmd, options.Stdin, options.Stdout, TerminalSizeChannel(options.TerminalSizeQueue))
	} else {
		execCmd.Stdin = options.Stdin
		execCmd.Stdout = options.Stdout
		execCmd.Stderr = options.Stderr
		return execCmd.Start()
	}
}

// TerminalSizeChannel convert size queue to size channel
func TerminalSizeChannel(queue remotecommand.TerminalSizeQueue) <-chan remotecommand.TerminalSize {
	ret := make(chan remotecommand.TerminalSize)
	go func() {
		for next := queue.Next(); next != nil; {
			ret <- *next
		}
	}()
	return ret
}
