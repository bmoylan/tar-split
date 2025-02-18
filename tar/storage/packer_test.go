package storage

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"
)

func TestDuplicateFail(t *testing.T) {
	e := []Entry{
		{
			Type:    FileType,
			Name:    "./hurr.txt",
			Payload: []byte("abcde"),
		},
		{
			Type:    FileType,
			Name:    "./hurr.txt",
			Payload: []byte("deadbeef"),
		},
		{
			Type:    FileType,
			Name:    "hurr.txt", // slightly different path, same file though
			Payload: []byte("deadbeef"),
		},
	}
	b := bytes.NewBuffer(nil)

	jp := NewJSONPacker(b)
	if _, err := jp.AddEntry(e[0]); err != nil {
		t.Error(err)
	}
	if _, err := jp.AddEntry(e[1]); err != ErrDuplicatePath {
		t.Errorf("expected failure on duplicate path")
	}
	if _, err := jp.AddEntry(e[2]); err != ErrDuplicatePath {
		t.Errorf("expected failure on duplicate path")
	}
}

func TestJSONPackerUnpacker(t *testing.T) {
	e := []Entry{
		{
			Type:    SegmentType,
			Payload: []byte("how"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("y'all"),
		},
		{
			Type:    FileType,
			Name:    "./hurr.txt",
			Payload: []byte("deadbeef"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("doin"),
		},
	}

	b := bytes.NewBuffer(nil)

	func() {
		jp := NewJSONPacker(b)
		for i := range e {
			if _, err := jp.AddEntry(e[i]); err != nil {
				t.Error(err)
			}
		}
	}()

	// >> packer_test.go:43: uncompressed: 266
	//t.Errorf("uncompressed: %d", len(b.Bytes()))

	b = bytes.NewBuffer(b.Bytes())
	entries := Entries{}
	func() {
		jup := NewJSONUnpacker(b)
		for {
			entry, err := jup.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Error(err)
			}
			entries = append(entries, *entry)
			t.Logf("got %#v", entry)
		}
	}()
	if len(entries) != len(e) {
		t.Errorf("expected %d entries, got %d", len(e), len(entries))
	}
}

// you can use a compress Reader/Writer and make nice savings.
//
// For these two tests that are using the same set, it the difference of 266
// bytes uncompressed vs 138 bytes compressed.
func TestGzip(t *testing.T) {
	e := []Entry{
		{
			Type:    SegmentType,
			Payload: []byte("how"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("y'all"),
		},
		{
			Type:    FileType,
			Name:    "./hurr.txt",
			Payload: []byte("deadbeef"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("doin"),
		},
	}

	b := bytes.NewBuffer(nil)
	gzW := gzip.NewWriter(b)
	jp := NewJSONPacker(gzW)
	for i := range e {
		if _, err := jp.AddEntry(e[i]); err != nil {
			t.Error(err)
		}
	}
	_ = gzW.Close()

	// >> packer_test.go:99: compressed: 138
	//t.Errorf("compressed: %d", len(b.Bytes()))

	b = bytes.NewBuffer(b.Bytes())
	gzR, err := gzip.NewReader(b)
	if err != nil {
		t.Fatal(err)
	}
	entries := Entries{}
	func() {
		jup := NewJSONUnpacker(gzR)
		for {
			entry, err := jup.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Error(err)
			}
			entries = append(entries, *entry)
			t.Logf("got %#v", entry)
		}
	}()
	if len(entries) != len(e) {
		t.Errorf("expected %d entries, got %d", len(e), len(entries))
	}
}

func BenchmarkGetPut(b *testing.B) {
	e := []Entry{
		{
			Type:    SegmentType,
			Payload: []byte("how"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("y'all"),
		},
		{
			Type:    FileType,
			Name:    "./hurr.txt",
			Payload: []byte("deadbeef"),
		},
		{
			Type:    SegmentType,
			Payload: []byte("doin"),
		},
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				fh, err := os.CreateTemp("", "tar-split.")
				if err != nil {
					b.Fatal(err)
				}
				defer func() {
					_ = fh.Close()
					_ = os.Remove(fh.Name())
				}()

				jp := NewJSONPacker(fh)
				for i := range e {
					if _, err := jp.AddEntry(e[i]); err != nil {
						b.Fatal(err)
					}
				}
				_ = fh.Sync()

				up := NewJSONUnpacker(fh)
				for {
					_, err := up.Next()
					if err != nil {
						if err == io.EOF {
							break
						}
						b.Fatal(err)
					}
				}

			}()
		}
	})
}
