package netframe

// Provide dial Optional Config Parameters

type dialOptions struct {
	retryInterval int //connect重试间隔, 为0时不自动重试
	dialTimeout   int //connect的超时时长,秒级
}

type DialOption interface {
	apply(*dialOptions)
}

type funcDialOption struct {
	f func(*dialOptions)
}

func (fdo *funcDialOption) apply(do *dialOptions) {
	fdo.f(do)
}

func newFuncDialOption(f func(*dialOptions)) *funcDialOption {
	return &funcDialOption{
		f: f,
	}
}

func WithRetryInterval(n int) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.retryInterval = n
	})
}

func WithDialTimeout(n int) DialOption {
	return newFuncDialOption(func(do *dialOptions) {
		do.dialTimeout = n
	})
}

func defaultDialOptions() dialOptions {
	return dialOptions{
		retryInterval: 3,
		dialTimeout:   5,
	}
}
