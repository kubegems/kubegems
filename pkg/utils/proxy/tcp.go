package proxy

import (
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"kubegems.io/kubegems/pkg/log"
)

// Proxy forwards a TCP request to a TCP service.
type Proxy struct {
	address          string
	target           *net.TCPAddr
	terminationDelay time.Duration
	refreshTarget    bool
}

// NewTCPProxy creates a new Proxy.
func NewTCPProxy(target string, terminationDelay time.Duration) (*Proxy, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		return nil, err
	}

	// enable the refresh of the target only if the address in an IP
	refreshTarget := false
	if host, _, err := net.SplitHostPort(target); err == nil && net.ParseIP(host) == nil {
		refreshTarget = true
	}

	return &Proxy{
		address:          target,
		target:           tcpAddr,
		refreshTarget:    refreshTarget,
		terminationDelay: terminationDelay,
	}, nil
}

// ServeTCP forwards the connection to a service.
func (p *Proxy) ServeConn(conn net.Conn) error {
	defer conn.Close()

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return errors.New("not a tcp conn")
	}
	if p.refreshTarget {
		tcpAddr, err := net.ResolveTCPAddr("tcp", p.address)
		if err != nil {
			return err
		}
		p.target = tcpAddr
	}

	connBackend, err := net.DialTCP("tcp", nil, p.target)
	if err != nil {
		return err
	}
	// set socket SO_REUSEADDR
	setSocketReuse(connBackend)

	// maybe not needed, but just in case
	defer connBackend.Close()

	if err := CopyDuplex(tcpConn, connBackend, p.terminationDelay); err != nil {
		return err
	}
	return nil
}

func connCopy(dst, src *net.TCPConn, errCh chan error, terminationDelay time.Duration) {
	_, err := io.Copy(dst, src)
	errCh <- err

	if err := dst.CloseWrite(); err != nil {
		log.Debugf("Error while terminating connection: %v", err)
		return
	}

	if terminationDelay >= 0 {
		if err := dst.SetReadDeadline(time.Now().Add(terminationDelay)); err != nil {
			log.Debugf("Error while setting deadline: %v", err)
		}
	}
}

func setSocketReuse(conn syscall.Conn) {
	sysconn, err := conn.SyscallConn()
	if err != nil {
		return
	}
	_ = sysconn.Control(
		func(fd uintptr) {
			if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
				return
			}
			if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
				return
			}
		},
	)
}

func CopyDuplex(a, b net.Conn, terminationDelay time.Duration) error {
	src, ok := a.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("not a tcp connection")
	}
	dst, ok := b.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("not a tcp connection")
	}

	errChan := make(chan error)

	go connCopy(src, dst, errChan, terminationDelay)
	go connCopy(dst, src, errChan, terminationDelay)

	err := <-errChan
	if err != nil {
		log.Debugf("Error during connection: %v", err)
	}

	<-errChan
	return err
}
