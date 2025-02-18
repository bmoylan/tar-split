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

// CommandDisasm provides the disasm command.
func CommandDisasm(c *cli.Context) {
	if len(c.Args()) != 1 {
		logrus.Fatalf("please specify tar to be disabled <NAME|->")
	}
	if len(c.String("output")) == 0 {
		logrus.Fatalf("--output filename must be set")
	}

	// Set up the tar input stream
	var inputStream io.Reader
	if c.Args()[0] == "-" {
		inputStream = os.Stdin
	} else {
		fh, err := os.Open(c.Args()[0])
		if err != nil {
			logrus.Fatal(err)
		}
		defer safeClose(fh)
		inputStream = fh
	}

	// Set up the metadata storage
	mf, err := os.OpenFile(c.String("output"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		logrus.Fatal(err)
	}
	defer safeClose(mf)
	mfz := gzip.NewWriter(mf)
	defer safeClose(mfz)
	metaPacker := storage.NewJSONPacker(mfz)

	// we're passing nil here for the file putter, because the ApplyDiff will
	// handle the extraction of the archive
	its, err := asm.NewInputTarStream(inputStream, metaPacker, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	var out io.Writer
	if c.Bool("no-stdout") {
		out = io.Discard
	} else {
		out = os.Stdout
	}
	i, err := io.Copy(out, its)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("created %s from %s (read %d bytes)", c.String("output"), c.Args()[0], i)
}
