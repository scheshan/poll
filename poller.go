package poll

type Flag uint64

func (t Flag) CanRead() bool {
	return t&1 > 0
}

func (t Flag) CanWrite() bool {
	return t&2 > 0
}

type Callback func(fd int, flag Flag) error
