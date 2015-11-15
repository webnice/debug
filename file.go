package debug

import (
	"errors"
	"os"
	"path/filepath"
)

// FileSave Save string to file. Create file if not exist and append file if exist. If file name is not specified, create file name is the same program name and .txt at extension
func FileSave(body []byte, file, path string) (err error) {
	const ext string = `.txt`
	var dir string
	var fileName string
	var fh *os.File
	var pi os.FileInfo

	if path == "" {
		dir, err = os.Getwd()
		if err != nil {
			return
		}
		path = dir
	} else {
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
			err = errors.New("Incorrect specified path. '" + path + "' is a file")
			return
		}
	}

	if file == "" {
		file = filepath.Base(os.Args[0]) + ext
	}

	fileName = path + string(os.PathSeparator) + file
	fh, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	_, err = fh.Write(body)
	if err != nil {
		return
	}
	err = fh.Close()
	return
}
