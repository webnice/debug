package debug

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileSave Save string to file. Create file if not exist and append file if exist. If file name is not specified, create file name is the same program name and .txt at extension
func FileSave(body []byte, file, path string) (err error) {
	const ext string = `.txt`
	var (
		fileName string
		fh       *os.File
		pi       os.FileInfo
	)

	if path, err = os.Getwd(); err != nil {
		return
	}
	if path != "" {
		pi, err = os.Stat(path)
		if err != nil && os.IsNotExist(err) == false {
			return
		}
		if err == nil {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return
			}
		}
		if pi.IsDir() == false {
			err = fmt.Errorf("incorrect specified path %q is a file", path)
			return
		}
	}
	if file == "" {
		file = filepath.Base(os.Args[0]) + ext
	}
	fileName = path + string(os.PathSeparator) + file
	if fh, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
		return
	}
	if _, err = fh.Write(body); err != nil {
		return
	}
	err = fh.Close()

	return
}
