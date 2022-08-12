package storage

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestGetter(t *testing.T) {
	fgp := NewBufferFileGetPutter()
	files := map[string]map[string][]byte{
		"file1.txt": {"foo": []byte{60, 60, 48, 48, 0, 0, 0, 0}},
		"file2.txt": {"bar": []byte{45, 196, 22, 240, 0, 0, 0, 0}},
	}
	for n, b := range files {
		for body, sum := range b {
			_, csum, err := fgp.Put(n, bytes.NewBufferString(body))
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(csum, sum) {
				t.Errorf("checksum: expected 0x%x; got 0x%x", sum, csum)
			}
		}
	}
	for n, b := range files {
		for body := range b {
			r, err := fgp.Get(&Entry{Name: n})
			if err != nil {
				t.Error(err)
			}
			buf, err := io.ReadAll(r)
			if err != nil {
				t.Error(err)
			}
			if body != string(buf) {
				t.Errorf("expected %q, got %q", body, string(buf))
			}
		}
	}
}

func TestChecksumGetPutter(t *testing.T) {
	files := []struct {
		Entry
		Body string
	}{
		{
			Entry: Entry{
				Type:    FileType,
				Name:    "file1.txt",
				Payload: []byte{60, 60, 48, 48, 0, 0, 0, 0},
			},
			Body: "foo",
		},
		{
			Entry: Entry{
				Type:    FileType,
				Name:    "file2.txt",
				Payload: []byte{45, 196, 22, 240, 0, 0, 0, 0},
			},
			Body: "bar",
		},
		{
			Entry: Entry{
				Type:    FileType,
				Name:    "file3.txt",
				Payload: []byte{32, 68, 22, 240, 0, 0, 0, 0},
			},
			Body: "baz",
		},
		{
			Entry: Entry{
				Type:    FileType,
				Name:    "file4.txt",
				Payload: []byte{48, 9, 150, 240, 0, 0, 0, 0},
			},
			Body: "bif",
		},
	}

	fp := NewChecksumFileGetter(t.TempDir())
	for _, file := range files {
		_, csum, err := fp.Put(file.Name, bytes.NewBufferString(file.Body))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(csum, file.Payload) {
			t.Errorf("checksum on %q: expected %x; got %x", file.Name, file.Payload, csum)
		}
	}

	for _, file := range files {
		r, err := fp.Get(&file.Entry)
		if err != nil {
			_ = r.Close()
			t.Fatal(err)
		}
		out, err := io.ReadAll(r)
		if err != nil {
			_ = r.Close()
			t.Fatal(err)
		}
		_ = r.Close()

		if string(out) != file.Body {
			t.Errorf("body on %q: expected %s; got %s", file.Name, file.Payload, out)
		}
	}
}

func BenchmarkPutter(b *testing.B) {
	files := []string{
		strings.Repeat("foo", 1000),
		strings.Repeat("bar", 1000),
		strings.Repeat("baz", 1000),
		strings.Repeat("fooz", 1000),
		strings.Repeat("vbatts", 1000),
		strings.Repeat("systemd", 1000),
	}
	for i := 0; i < b.N; i++ {
		fgp := NewBufferFileGetPutter()
		for n, body := range files {
			if _, _, err := fgp.Put(fmt.Sprintf("%d", n), bytes.NewBufferString(body)); err != nil {
				b.Fatal(err)
			}
		}
	}
}
