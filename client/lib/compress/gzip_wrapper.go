package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
)

type gzipWriter struct {
	w     io.Writer
	level int
}

func newGzipWriter(w io.Writer, level int) *gzipWriter {
	return &gzipWriter{
		w:     w,
		level: level,
	}
}

func (gw gzipWriter) Write(p []byte) (int, error) {
	buf := new(bytes.Buffer)
	written, err := func() (int, error) {
		w, err := gzip.NewWriterLevel(buf, gw.level)
		if err != nil {
			return 0, err
		}

		var written int
		for {
			n, err := w.Write(p)
			if err != nil {
				return n, err
			}
			if n == 0 {
				break
			}
			p = p[n:]
			written += n
		}

		if err := w.Flush(); err != nil {
			return written, err
		}
		return written, w.Close()
	}()
	if err != nil {
		return written, err
	}
	return gw.w.Write(buf.Bytes())
}

type gzipReader struct {
	*gzip.Reader
	rd io.Reader
}

func newGzipReader(rd io.Reader) (*gzipReader, error) {
	/*gr, err := gzip.NewReader(rd)
	if err != nil {
		return nil, err
	}*/
	return &gzipReader{
		//Reader: gr,
		rd: rd,
	}, nil
}

func (gr gzipReader) ReadFull(data []byte) ([]byte, error) {
	br := bytes.NewReader(data)
	rd, err := gzip.NewReader(br)
	if err != nil {
		return nil, err
	}
	defer rd.Close()

	b, err := ioutil.ReadAll(rd)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return b, nil
}
