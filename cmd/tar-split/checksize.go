package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/bmoylan/tar-split/tar/asm"
	"github.com/bmoylan/tar-split/tar/storage"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// CommandChecksize provides the checksize command.
func CommandChecksize(c *cli.Context) {
	if len(c.Args()) == 0 {
		logrus.Fatalf("please specify tar archives to check ('-' will check stdin)")
	}
	for _, arg := range c.Args() {
		fh, err := os.Open(arg)
		if err != nil {
			log.Fatal(err)
		}
		defer safeClose(fh)
		fi, err := fh.Stat()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("inspecting %q (size %dk)\n", fh.Name(), fi.Size()/1024)

		packFh, err := os.CreateTemp("", "packed.")
		if err != nil {
			log.Fatal(err)
		}
		defer safeClose(packFh)
		if !c.Bool("work") {
			defer func() { _ = os.Remove(packFh.Name()) }()
		} else {
			fmt.Printf(" -- working file preserved: %s\n", packFh.Name())
		}

		sp := storage.NewJSONPacker(packFh)
		fp := storage.NewDiscardFilePutter()
		dissam, err := asm.NewInputTarStream(fh, sp, fp)
		if err != nil {
			log.Fatal(err)
		}

		var num int
		tr := tar.NewReader(dissam)
		for {
			_, err = tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatal(err)
			}
			num++
			if _, err := io.Copy(ioutil.Discard, tr); err != nil {
				log.Fatal(err)
			}
		}
		fmt.Printf(" -- number of files: %d\n", num)

		if err := packFh.Sync(); err != nil {
			log.Fatal(err)
		}

		fi, err = packFh.Stat()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" -- size of metadata uncompressed: %dk\n", fi.Size()/1024)

		gzPackFh, err := ioutil.TempFile("", "packed.gz.")
		if err != nil {
			log.Fatal(err)
		}
		defer safeClose(gzPackFh)
		if !c.Bool("work") {
			defer func() { _ = os.Remove(gzPackFh.Name()) }()
		}

		gzWrtr := gzip.NewWriter(gzPackFh)

		if _, err := packFh.Seek(0, 0); err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(gzWrtr, packFh); err != nil {
			log.Fatal(err)
		}
		safeClose(gzWrtr)

		if err := gzPackFh.Sync(); err != nil {
			log.Fatal(err)
		}

		fi, err = gzPackFh.Stat()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" -- size of gzip compressed metadata: %dk\n", fi.Size()/1024)
	}
}
