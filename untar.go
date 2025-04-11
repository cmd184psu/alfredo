package alfredo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		// Clean the path and ensure it is within the destination directory
		cleanedPath := filepath.Clean(header.Name)
		if strings.Contains(cleanedPath, "..") {
			return fmt.Errorf("invalid file path: %s", cleanedPath)
		}
		target := filepath.Join(dst, cleanedPath)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

func Targz(sourceDir string, outputFilename string, leaveout string) error {
	// Create the output file
	VerbosePrintln("start of targz")
	output, err := os.Create(outputFilename)
	if err != nil {
		VerbosePrintln("Failed to create file")
		return err
	}
	defer output.Close()

	// Create a gzip writer for the output file
	gzipWriter := gzip.NewWriter(output)
	defer gzipWriter.Close()

	// Create a tar writer for the gzip writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	//sort.Strings(leaveout)

	// Walk through the source directory and add files to the tar archive
	err = filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if StringContainsInCSV(filePath, leaveout) {
			return nil
		}

		//res := sort.SearchStrings(leaveout, sourceDir)

		if err != nil {
			VerbosePrintln("Failed to take a walk")
			return err
		}

		// Create a tar header for the file
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			VerbosePrintln("Failed to obtain header")
			return err
		}

		// Modify the header name to be relative to the source directory
		relPath, _ := filepath.Rel(sourceDir, filePath)
		header.Name = relPath

		// Write the header to the tar archive
		if err := tarWriter.WriteHeader(header); err != nil {
			VerbosePrintln("Failed to write header")
			return err
		}

		// If the file is not a directory, write its contents to the archive
		if !info.IsDir() {
			if !info.Mode().IsRegular() { //nothing more to do for non-regular
				return nil
			}

			file, err := os.Open(filePath)
			if err != nil {
				VerbosePrintln("problem opening " + filePath)

				return err
			}
			defer file.Close()
			var d int64
			//			d, err = io.Copy(tarWriter, file)
			var buf []byte
			d, err = io.CopyBuffer(tarWriter, file, buf)
			VerbosePrintln(fmt.Sprintf("d=%d", d))
			if err != nil {
				VerbosePrintln("problem copying data via io.Copy(...,...) ")

				return err
			}
		}

		return nil
	})

	VerbosePrintln("end of targz")

	return err
}
