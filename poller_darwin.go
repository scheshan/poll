package poll

import (
	"golang.org/x/sys/unix"
)

type Poller struct {
	kq int
	t  int64
	ts *unix.Timespec
	ev []unix.Kevent_t
}

func (t *Poller) Add(fd int) error {
	return t.mod(fd, unix.EV_ADD, unix.EVFILT_READ)
}

func (t *Poller) Delete(fd int) error {
	return t.mod(fd, unix.EV_DELETE|unix.EV_ONESHOT, unix.EVFILT_READ, unix.EVFILT_WRITE)
}

func (t *Poller) EnableWrite(fd int) error {
	return t.mod(fd, unix.EV_ADD, unix.EVFILT_WRITE)
}

func (t *Poller) DisableWrite(fd int) error {
	return t.mod(fd, unix.EV_DELETE|unix.EV_ONESHOT, unix.EVFILT_WRITE)
}

func (t *Poller) Wait(waitMs int64, cb Callback) error {
	if t.t != waitMs {
		t.t = waitMs

		t.ts.Sec = waitMs / 1000
		t.ts.Nsec = (waitMs % 1000) * 1000000
	}

	n, err := unix.Kevent(t.kq, nil, t.ev, t.ts)
	if err != nil {
		return err
	}

	if cb != nil {
		for i := 0; i < n; i++ {
			flag := Flag(0)
			if t.ev[i].Filter == unix.EVFILT_READ {
				flag |= 1
			} else if t.ev[i].Filter == unix.EVFILT_WRITE {
				flag |= 2
			}

			cb(int(t.ev[i].Ident), flag)
		}
	}

	if n == len(t.ev) {
		t.incrEv(n << 1)
	}

	return nil
}

func (t *Poller) Wakeup() error {
	_, err := unix.Kevent(t.kq, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

func (t *Poller) Close() {
	if t.kq > 0 {
		unix.Close(t.kq)
		t.kq = 0
	}
}

func (t *Poller) mod(fd int, flags uint16, filters ...int16) error {
	changes := make([]unix.Kevent_t, len(filters))

	for i, filter := range filters {
		changes[i].Ident = uint64(fd)
		changes[i].Filter = filter
		changes[i].Flags = flags
	}

	_, err := unix.Kevent(t.kq, changes, nil, nil)
	return err
}

func (t *Poller) incrEv(size int) {
	t.ev = make([]unix.Kevent_t, size)
}

func NewPoller() (*Poller, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}

	p := &Poller{}
	p.kq = kq

	err = p.mod(0, unix.EV_ADD|unix.EV_CLEAR, unix.EVFILT_USER)
	if err != nil {
		p.Close()
		return nil, err
	}

	p.ts = &unix.Timespec{}
	p.incrEv(1024)

	return p, nil
}
