package storage

import (
	"bytes"
	"encoding/hex"
	"errors"
	"hash/crc64"
	"io"
	"os"
	"path/filepath"
)

// FileGetter is the interface for getting a stream of a file payload,
// addressed by Entry. Presumably, the names will be scoped to relative
// file paths.
type FileGetter interface {
	// Get returns a stream for the provided Entry
	Get(entry *Entry) (output io.ReadCloser, err error)
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

func (pfg pathFileGetter) Get(entry *Entry) (io.ReadCloser, error) {
	return os.Open(filepath.Join(pfg.root, entry.GetName()))
}

type bufferFileGetPutter struct {
	files map[string][]byte
}

func (bfgp bufferFileGetPutter) Get(entry *Entry) (io.ReadCloser, error) {
	name := entry.GetName()
	if _, ok := bfgp.files[name]; !ok {
		return nil, errors.New("no such file")
	}
	b := bytes.NewBuffer(bfgp.files[name])
	return io.NopCloser(b), nil
}

func (bfgp *bufferFileGetPutter) Put(name string, r io.Reader) (int64, []byte, error) {
	crc := crc64.New(CRCTable)
	buf := bytes.NewBuffer(nil)
	cw := io.MultiWriter(crc, buf)
	i, err := io.Copy(cw, r)
	if err != nil {
		return 0, nil, err
	}
	bfgp.files[name] = buf.Bytes()
	return i, crc.Sum(nil), nil
}

// NewBufferFileGetPutter is a simple in-memory FileGetPutter
//
// Implication is this is memory intensive...
// Probably best for testing or light weight cases.
func NewBufferFileGetPutter() FileGetPutter {
	return &bufferFileGetPutter{
		files: map[string][]byte{},
	}
}

type checksumFileGetPutter struct {
	root string
}

// NewChecksumFileGetter returns a FileGetter that is for files stored by crc64 checksum.
func NewChecksumFileGetter(relpath string) FileGetPutter {
	return &checksumFileGetPutter{root: relpath}
}

func (cfg checksumFileGetPutter) Get(entry *Entry) (io.ReadCloser, error) {
	if entry.Type == SegmentType {
		return os.Open(filepath.Join(cfg.root, entry.GetName()))
	}
	return os.Open(filepath.Join(cfg.root, hex.EncodeToString(entry.Payload)))
}

func (cfg checksumFileGetPutter) Put(_ string, r io.Reader) (int64, []byte, error) {
	tmp, err := os.CreateTemp(cfg.root, "checksumFileGetPutter-*")
	if err != nil {
		return 0, nil, err
	}
	i, checksum, err := copyWithChecksum(tmp, r)
	if err != nil {
		return 0, nil, err
	}
	checksumPath := filepath.Join(cfg.root, hex.EncodeToString(checksum))
	if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
		// checksum-addressed file does not yet exist
		if err := os.Rename(tmp.Name(), checksumPath); err != nil {
			return 0, nil, err
		}
	} else {
		// checksum-addressed file already exists
		if err := os.Remove(tmp.Name()); err != nil {
			return 0, nil, err
		}
	}
	return i, checksum, nil
}

// NewDiscardFilePutter is a bit bucket FilePutter
func NewDiscardFilePutter() FilePutter {
	return &bitBucketFilePutter{}
}

type bitBucketFilePutter struct {
	buffer [32 * 1024]byte // 32 kB is the buffer size currently used by io.Copy, as of August 2021.
}

func (bbfp *bitBucketFilePutter) Put(name string, r io.Reader) (int64, []byte, error) {
	crc := crc64.New(CRCTable)
	i, err := io.CopyBuffer(crc, r, bbfp.buffer[:])
	return i, crc.Sum(nil), err
}

// CRCTable is the default table used for crc64 sum calculations
var CRCTable = crc64.MakeTable(crc64.ISO)

func copyWithChecksum(w io.WriteCloser, r io.Reader) (int64, []byte, error) {
	crc := crc64.New(CRCTable)
	cw := io.MultiWriter(crc, w)
	defer func() { _ = w.Close() }()
	i, err := io.Copy(cw, r)
	if err != nil {
		return 0, nil, err
	}
	return i, crc.Sum(nil), nil
}
