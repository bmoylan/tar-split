package main

import (
	"compress/gzip"
	"io"
	"os"

	"github.com/bmoylan/tar-split/tar/asm"
	"github.com/bmoylan/tar-split/tar/storage"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// CommandAsm provides the asm command.
func CommandAsm(c *cli.Context) {
	if len(c.Args()) > 0 {
		logrus.Warnf("%d additional arguments passed are ignored", len(c.Args()))
	}
	if len(c.String("input")) == 0 {
		logrus.Fatalf("--input filename must be set")
	}
	if len(c.String("output")) == 0 {
		logrus.Fatalf("--output filename must be set ([FILENAME|-])")
	}
	if len(c.String("path")) == 0 {
		logrus.Fatalf("--path must be set")
	}

	var outputStream io.Writer
	if c.String("output") == "-" {
		outputStream = os.Stdout
	} else {
		fh, err := os.Create(c.String("output"))
		if err != nil {
			logrus.Fatal(err)
		}
		defer safeClose(fh)
		outputStream = fh
	}

	if c.Bool("compress") {
		zipper := gzip.NewWriter(outputStream)
		defer safeClose(zipper)
		outputStream = zipper
	}

	// Get the tar metadata reader
	mf, err := os.Open(c.String("input"))
	if err != nil {
		logrus.Fatal(err)
	}
	defer safeClose(mf)
	mfz, err := gzip.NewReader(mf)
	if err != nil {
		logrus.Fatal(err)
	}
	defer safeClose(mfz)

	metaUnpacker := storage.NewJSONUnpacker(mfz)
	// XXX maybe get the absolute path here
	fileGetter := storage.NewPathFileGetter(c.String("path"))

	ots := asm.NewOutputTarStream(fileGetter, metaUnpacker)
	defer safeClose(ots)
	i, err := io.Copy(outputStream, ots)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("created %s from %s and %s (wrote %d bytes)", c.String("output"), c.String("path"), c.String("input"), i)
}

func safeClose(closer io.Closer) {
	if err := closer.Close(); err != nil {
		logrus.Error(err)
	}
}
