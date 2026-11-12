package debug

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileSave сохраняет данные в файл: создаёт файл, если он не существует, дописывает данные
// в конец существующего. Если имя файла не указано, используется имя программы с расширением .txt.
// Если путь не указан, используется текущая рабочая директория; отсутствующие директории создаются.
func FileSave(body []byte, file, path string) (err error) {
	const (
		tplExt    string = ".txt"
		errNotDir string = "указанный путь %q не является директорией"
	)
	var (
		fi   os.FileInfo
		fh   *os.File
		cerr error
	)

	if path == "" {
		if path, err = getwd(); err != nil {
			return
		}
	}
	// Существующая директория используется как есть, отсутствующая создаётся.
	switch fi, err = os.Stat(path); {
	case err == nil:
		if !fi.IsDir() {
			return fmt.Errorf(errNotDir, path)
		}
	case os.IsNotExist(err):
		if err = mkdirAll(path, 0o755); err != nil {
			return
		}
	default:
		return
	}
	// Пустое имя файла: используется имя программы с расширением .txt.
	if file == "" {
		file = filepath.Base(os.Args[0]) + tplExt
	}
	// Данные дописываются в конец файла.
	if fh, err = os.OpenFile(filepath.Join(path, file), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return
	}
	defer func() {
		if cerr = fh.Close(); err == nil {
			err = cerr
		}
	}()
	_, err = fh.Write(body)

	return
}
