package asm

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/bmoylan/tar-split/tar/storage"
)

// NewOutputTarStream returns an io.ReadCloser that is an assembled tar archive
// stream.
//
// It takes a storage.FileGetter, for mapping the file payloads that are to be read in,
// and a storage.Unpacker, which has access to the rawbytes and file order
// metadata. With the combination of these two items, a precise assembled Tar
// archive is possible.
func NewOutputTarStream(fg storage.FileGetter, up storage.Unpacker) io.ReadCloser {
	// ... Since these are interfaces, this is possible, so let's not have a nil pointer
	if fg == nil || up == nil {
		return nil
	}
	pr, pw := io.Pipe()
	go func() {
		err := WriteOutputTarStream(fg, up, pw)
		_ = pw.CloseWithError(err)
	}()
	return pr
}

// WriteOutputTarStream writes assembled tar archive to a writer.
func WriteOutputTarStream(fg storage.FileGetter, up storage.Unpacker, w io.Writer) error {
	// ... Since these are interfaces, this is possible, so let's not have a nil pointer
	if fg == nil || up == nil {
		return nil
	}
	var copyBuffer []byte
	defer byteBufferPool.Put(copyBuffer)
	var crcHash hash.Hash
	var crcSum []byte
	var multiWriter io.Writer
	for {
		entry, err := up.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch entry.Type {
		case storage.SegmentType:
			if _, err := w.Write(entry.Payload); err != nil {
				return err
			}
		case storage.FileType:
			if entry.Size == 0 {
				continue
			}
			fh, err := fg.Get(entry)
			if err != nil {
				return err
			}
			if crcHash == nil {
				crcHash = storage.NewHash()
				crcSum = make([]byte, crcHash.Size())
				multiWriter = io.MultiWriter(w, crcHash)
				copyBuffer = byteBufferPool.Get().([]byte)
			} else {
				crcHash.Reset()
			}

			if _, err := io.CopyBuffer(multiWriter, fh, copyBuffer); err != nil {
				_ = fh.Close()
				return err
			}

			if !bytes.Equal(crcHash.Sum(crcSum[:0]), entry.Payload) {
				// I would rather this be a comparable ErrInvalidChecksum or such,
				// but since it's coming through the PipeReader, the context of
				// _which_ file would be lost...
				_ = fh.Close()
				return fmt.Errorf("file integrity checksum failed for %q", entry.GetName())
			}
			_ = fh.Close()
		}
	}
}

var byteBufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}
