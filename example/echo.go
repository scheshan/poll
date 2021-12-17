package main

import (
	"github.com/scheshan/poll"
	"golang.org/x/sys/unix"
	"log"
	"sync"
)

func main() {
	srv, err := newEchoServer(16379)
	if err != nil {
		log.Fatal(err)
	}

	err = srv.run()
	if err != nil {
		log.Fatal(err)
	}

}

type echoServer struct {
	port   int
	lfd    int
	poller *poll.Poller
	wg     *sync.WaitGroup
	buf    []byte
}

func (t *echoServer) run() error {
	if err := t.bindAndListen(); err != nil {
		unix.Close(t.lfd)
		t.poller.Close()

		log.Fatal(err)
	}

	t.loop()
	return nil
}

func (t *echoServer) bindAndListen() error {
	lfd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	t.lfd = lfd

	if err = unix.SetNonblock(lfd, true); err != nil {
		return err
	}

	if err = unix.SetsockoptInt(lfd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return err
	}

	if err = unix.SetsockoptInt(lfd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return err
	}

	addr := &unix.SockaddrInet4{Port: t.port}
	if err = unix.Bind(lfd, addr); err != nil {
		return err
	}

	if err = unix.Listen(lfd, 1024); err != nil {
		return err
	}
	if err = t.poller.Add(lfd); err != nil {
		return err
	}

	return nil
}

func (t *echoServer) loop() {
	for {
		t.poller.Wait(1000, t.pollCallback)
	}
}

func (t *echoServer) pollCallback(fd int, flag poll.Flag) {
	if fd == t.lfd {
		conn, _, err := unix.Accept(t.lfd)
		unix.SetNonblock(conn, true)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				return
			}

			log.Fatal(err)
		}

		t.poller.Add(conn)
	} else {
		n, err := unix.Read(fd, t.buf)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EINTR {
				return
			}

			unix.Close(fd)
			return
		}

		if n == 0 {
			unix.Close(fd)
			return
		}

		unix.Write(fd, t.buf[:n])
	}
}

func newEchoServer(port int) (*echoServer, error) {
	p, err := poll.NewPoller()
	if err != nil {
		return nil, err
	}

	srv := &echoServer{
		poller: p,
		port:   port,
		wg:     &sync.WaitGroup{},
		buf:    make([]byte, 4096),
	}

	return srv, nil
}
