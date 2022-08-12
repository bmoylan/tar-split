package storage

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// FileGetter is the interface for getting a stream of a file payload,
// addressed by name/filename. Presumably, the names will be scoped to relative
// file paths.
type FileGetter interface {
	// Get returns a stream for the provided file path
	Get(filename string) (output io.ReadCloser, err error)
}

// FilePutter is the interface for storing a stream of a file payload,
// addressed by name/filename.
type FilePutter interface {
	// Put returns the size of the stream received, and the crc64 checksum for
	// the provided stream
	Put(filename string, input io.Reader) (size int64, checksum []byte, err error)
}

// FileGetPutter is the interface that groups both Getting and Putting file
// payloads.
type FileGetPutter interface {
	FileGetter
	FilePutter
}

// NewPathFileGetter returns a FileGetter that is for files relative to path
// relpath.
func NewPathFileGetter(relpath string) FileGetter {
	return &pathFileGetter{root: relpath}
}

type pathFileGetter struct {
	root string
}

func (pfg pathFileGetter) Get(filename string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(pfg.root, filename))
}

type bufferFileGetPutter struct {
	files map[string][]byte
}

func (bfgp bufferFileGetPutter) Get(name string) (io.ReadCloser, error) {
	if _, ok := bfgp.files[name]; !ok {
		return nil, errors.New("no such file")
	}
	b := bytes.NewBuffer(bfgp.files[name])
	return &readCloserWrapper{b}, nil
}

func (bfgp *bufferFileGetPutter) Put(name string, r io.Reader) (int64, []byte, error) {
	hsh := NewHash()
	buf := bytes.NewBuffer(nil)
	cw := io.MultiWriter(hsh, buf)
	i, err := io.Copy(cw, r)
	if err != nil {
		return 0, nil, err
	}
	bfgp.files[name] = buf.Bytes()
	return i, hsh.Sum(nil), nil
}

type readCloserWrapper struct {
	io.Reader
}

func (w *readCloserWrapper) Close() error { return nil }

// NewBufferFileGetPutter is a simple in-memory FileGetPutter
//
// Implication is this is memory intensive...
// Probably best for testing or light weight cases.
func NewBufferFileGetPutter() FileGetPutter {
	return &bufferFileGetPutter{
		files: map[string][]byte{},
	}
}

// NewDiscardFilePutter is a bit bucket FilePutter
func NewDiscardFilePutter() FilePutter {
	return &bitBucketFilePutter{}
}

type bitBucketFilePutter struct {
	buffer [32 * 1024]byte // 32 kB is the buffer size currently used by io.Copy, as of August 2021.
}

func (bbfp *bitBucketFilePutter) Put(name string, r io.Reader) (int64, []byte, error) {
	hsh := NewHash()
	i, err := io.CopyBuffer(hsh, r, bbfp.buffer[:])
	return i, hsh.Sum(nil), err
}
