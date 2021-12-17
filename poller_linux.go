package poll

import "golang.org/x/sys/unix"

var wakeupData = []byte{1, 0, 0, 0, 0, 0, 0, 0}

type Poller struct {
	ep int
	ef int
	ev []unix.EpollEvent
}

func (t *Poller) Add(fd int) error {
	return t.mod(fd, unix.EPOLL_CTL_ADD, unix.EPOLLIN)
}

func (t *Poller) AddWrite(fd int) error {
	return t.mod(fd, unix.EPOLL_CTL_ADD, unix.EPOLLIN|unix.EPOLLOUT)
}

func (t *Poller) Delete(fd int) error {
	return t.mod(fd, unix.EPOLL_CTL_DEL, unix.EPOLLIN|unix.EPOLLOUT)
}

func (t *Poller) EnableWrite(fd int) error {
	return t.mod(fd, unix.EPOLL_CTL_MOD, unix.EPOLLIN|unix.EPOLLOUT)
}

func (t *Poller) DisableWrite(fd int) error {
	return t.mod(fd, unix.EPOLL_CTL_MOD, unix.EPOLLIN)
}

func (t *Poller) Wait(waitMs int64, cb Callback) error {
	n, err := unix.EpollWait(t.ep, t.ev, int(waitMs))
	if err != nil {
		return err
	}

	if cb != nil {
		for i := 0; i < n; i++ {
			flag := Flag(0)
			if t.ev[i].Events&unix.EPOLLIN > 0 {
				flag |= 1
			}
			if t.ev[i].Events&unix.EPOLLOUT > 0 {
				flag |= 2
			}

			cb(int(t.ev[i].Fd), flag)
		}
	}

	if n == len(t.ev) {
		t.incrEv(n << 1)
	}

	return nil
}

func (t *Poller) Wakeup() error {
	_, err := unix.Write(t.ef, wakeupData)
	return err
}

func (t *Poller) Close() {
	if t.ef > 0 {
		unix.Close(t.ef)
		t.ef = 0
	}
	if t.ep > 0 {
		unix.Close(t.ep)
		t.ep = 0
	}
}

func (t *Poller) mod(fd int, mode int, events uint32) error {
	ev := &unix.EpollEvent{}
	ev.Fd = int32(fd)
	ev.Events = events

	return unix.EpollCtl(t.ep, mode, fd, ev)
}

func (t *Poller) incrEv(size int) {
	t.ev = make([]unix.EpollEvent, size)
}

func NewPoller() (*Poller, error) {
	ep, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	p := &Poller{}
	p.ep = ep

	ef, err := unix.Eventfd(0, 0)
	if err != nil {
		p.Close()
		return nil, err
	}
	p.ef = ef

	err = p.Add(ef)
	if err != nil {
		p.Close()
		return nil, err
	}

	p.incrEv(1024)

	return p, nil
}
