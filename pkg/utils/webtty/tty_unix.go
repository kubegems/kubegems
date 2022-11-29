package webtty

import (
	"io"
	"os/exec"

	"github.com/creack/pty"
	"k8s.io/client-go/tools/remotecommand"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"kubegems.io/kubegems/pkg/log"
)

func TTYCmd(execCmd *exec.Cmd, masterin io.Reader, masterout io.Writer, resize <-chan remotecommand.TerminalSize) error {
	p, err := pty.Start(execCmd)
	if err != nil {
		return err
	}
	defer p.Close()

	kubecontainer.HandleResizing(resize, func(size remotecommand.TerminalSize) {
		if err := pty.Setsize(p, &pty.Winsize{Rows: size.Height, Cols: size.Width}); err != nil {
			log.Error(err, "unable to set terminal size")
		}
	})

	var stdinErr, stdoutErr error
	if masterin != nil {
		go func() { _, stdinErr = io.Copy(p, masterin) }()
	}
	if masterout != nil {
		go func() { _, stdoutErr = io.Copy(masterout, p) }()
	}
	err = execCmd.Wait()

	if stdinErr != nil {
		log.Error(err, "stdin copy error")
	}
	if stdoutErr != nil {
		log.Error(err, "stdout copy error")
	}
	return err
}
