package main

import (
	upTar "archive/tar"
	"io"
	"os"
	"testing"

	ourTar "github.com/vbatts/tar-split/archive/tar"
)

var testfile = "../../archive/tar/testdata/sparse-formats.tar"

func BenchmarkUpstreamTar(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		fh, err := os.Open(testfile)
		if err != nil {
			b.Fatal(err)
		}
		tr := upTar.NewReader(fh)
		for {
			_, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				safeClose(fh)
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, tr)
		}
		safeClose(fh)
	}
}

func BenchmarkOurTarNoAccounting(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		fh, err := os.Open(testfile)
		if err != nil {
			b.Fatal(err)
		}
		tr := ourTar.NewReader(fh)
		tr.RawAccounting = false // this is default, but explicit here
		for {
			_, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				safeClose(fh)
				b.Fatal(err)
			}
			if _, err := io.Copy(io.Discard, tr); err != nil {
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, tr)
		}
		safeClose(fh)
	}
}

func BenchmarkOurTarYesAccounting(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		fh, err := os.Open(testfile)
		if err != nil {
			b.Fatal(err)
		}
		tr := ourTar.NewReader(fh)
		tr.RawAccounting = true // This enables mechanics for collecting raw bytes
		for {
			_ = tr.RawBytes()
			_, err := tr.Next()
			_ = tr.RawBytes()
			if err != nil {
				if err == io.EOF {
					break
				}
				safeClose(fh)
				b.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, tr)
			_ = tr.RawBytes()
		}
		safeClose(fh)
	}
}
