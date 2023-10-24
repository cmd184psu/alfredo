package alfredo

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Gunzip(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	fmt.Println("want to create file: " + dst)
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, os.FileMode(int(0644)))
	if err != nil {
		return err
	}

	// copy over contents
	if _, err := io.Copy(f, gzr); err != nil {
		return err
	}

	// manually close here after each file operation; defering would cause each file close
	// to wait until all operations have completed.
	defer f.Close()
	return nil
}
