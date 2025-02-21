package mpb

import (
	"io"
	"io/ioutil"
	"time"
)

type proxyReader struct {
	io.ReadCloser
	bar *Bar
}

func (x *proxyReader) Read(p []byte) (int, error) {
	n, err := x.ReadCloser.Read(p)
	x.bar.IncrBy(n)
	if err == io.EOF {
		go x.bar.SetTotal(-1, true)
	}
	return n, err
}

type proxyWriterTo struct {
	io.ReadCloser // *proxyReader
	wt            io.WriterTo
	bar           *Bar
}

func (x *proxyWriterTo) WriteTo(w io.Writer) (int64, error) {
	n, err := x.wt.WriteTo(w)
	x.bar.IncrInt64(n)
	if err == io.EOF {
		go x.bar.SetTotal(-1, true)
	}
	return n, err
}

type ewmaProxyReader struct {
	io.ReadCloser // *proxyReader
	bar           *Bar
}

func (x *ewmaProxyReader) Read(p []byte) (int, error) {
	start := time.Now()
	n, err := x.ReadCloser.Read(p)
	if n > 0 {
		x.bar.DecoratorEwmaUpdate(time.Since(start))
	}
	return n, err
}

type ewmaProxyWriterTo struct {
	io.ReadCloser             // *ewmaProxyReader
	wt            io.WriterTo // *proxyWriterTo
	bar           *Bar
}

func (x *ewmaProxyWriterTo) WriteTo(w io.Writer) (int64, error) {
	start := time.Now()
	n, err := x.wt.WriteTo(w)
	if n > 0 {
		x.bar.DecoratorEwmaUpdate(time.Since(start))
	}
	return n, err
}

func newProxyReader(r io.Reader, bar *Bar) io.ReadCloser {
	rc := toReadCloser(r)
	rc = &proxyReader{rc, bar}

	if wt, isWriterTo := r.(io.WriterTo); bar.hasEwmaDecorators {
		rc = &ewmaProxyReader{rc, bar}
		if isWriterTo {
			rc = &ewmaProxyWriterTo{rc, wt, bar}
		}
	} else if isWriterTo {
		rc = &proxyWriterTo{rc, wt, bar}
	}
	return rc
}

func toReadCloser(r io.Reader) io.ReadCloser {
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}
	return ioutil.NopCloser(r)
}
