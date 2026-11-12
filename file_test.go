package debug

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Общие шаблоны сообщений тестов файлового вывода.
const (
	tplFileOut  string = "out.txt"
	errPrepFile string = "не удалось подготовить тестовый файл: %v"
	errBadBody  string = "неверное содержимое файла: %q"
)

// FileSave сохраняет файл в указанный путь и создаёт отсутствующие директории.
func TestFileSavePath(t *testing.T) {
	const (
		tplBody     string = "привет"
		errNoAtPath string = "файл не создан в указанном пути: %v"
	)
	var (
		dir  string
		sub  string
		body []byte
		err  error
	)

	dir = t.TempDir()
	sub = filepath.Join(dir, "sub", "dir")
	if err = FileSave([]byte(tplBody), tplFileOut, sub); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if body, err = os.ReadFile(filepath.Join(sub, tplFileOut)); err != nil {
		t.Fatalf(errNoAtPath, err)
	}
	if string(body) != tplBody {
		t.Errorf(errBadBody, body)
	}
}

// FileSave дописывает данные в конец существующего файла.
//
//goland:noinspection
func TestFileSaveAppend(t *testing.T) {
	//goland:noinspection SpellCheckingInspection
	const (
		tplOne    string = "раз"
		tplTwo    string = "два"
		tplBoth   string = "раздва"
		errNoFile string = "файл не создан: %v"
	)
	var (
		dir  string
		body []byte
		err  error
	)

	dir = t.TempDir()
	if err = FileSave([]byte(tplOne), tplFileOut, dir); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if err = FileSave([]byte(tplTwo), tplFileOut, dir); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	if body, err = os.ReadFile(filepath.Join(dir, tplFileOut)); err != nil {
		t.Fatalf(errNoFile, err)
	}
	if string(body) != tplBoth {
		t.Errorf(errBadBody, body)
	}
}

// FileSave с пустым именем файла использует имя программы с расширением .txt.
func TestFileSaveDefaultName(t *testing.T) {
	const (
		tplBody       string = "данные"
		tplExt        string = ".txt"
		errNoFileName string = "файл %q не создан: %v"
	)
	var (
		dir  string
		want string
		err  error
	)

	dir = t.TempDir()
	if err = FileSave([]byte(tplBody), "", dir); err != nil {
		t.Fatalf(errUnexpected, err)
	}
	want = filepath.Join(dir, filepath.Base(os.Args[0])+tplExt)
	if _, err = os.Stat(want); err != nil {
		t.Errorf(errNoFileName, want, err)
	}
}

// FileSave возвращает ошибку, если путь указывает на обычный файл.
func TestFileSavePathIsFile(t *testing.T) {
	const (
		tplBlocker  string = "файл.txt"
		errPathFile string = "ожидалась ошибка при передаче файла в качестве пути"
	)
	var (
		dir string
		f   string
		err error
	)

	dir = t.TempDir()
	f = filepath.Join(dir, tplBlocker)
	if err = os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf(errPrepFile, err)
	}
	if err = FileSave([]byte("данные"), tplFileOut, f); err == nil {
		t.Error(errPathFile)
	}
}

// FileSave возвращает ошибку getwd, если текущая директория недоступна.
func TestFileSaveGetwdError(t *testing.T) {
	const (
		errGetwdFail string = "сбой getwd"
		errNoGetwd   string = "ожидалась ошибка getwd"
	)
	var (
		orig func() (string, error)
		err  error
	)

	orig = getwd
	getwd = func() (string, error) { return "", errors.New(errGetwdFail) }
	defer func() { getwd = orig }()
	if err = FileSave([]byte("x"), tplFileOut, ""); err == nil {
		t.Error(errNoGetwd)
	}
}

// FileSave возвращает ошибку stat, если компонент пути не является директорией.
func TestFileSaveStatError(t *testing.T) {
	const (
		tplBlocker string = "файл"
		errNoStat  string = "ожидалась ошибка stat для пути через обычный файл"
	)
	var (
		dir string
		err error
	)

	dir = t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, tplBlocker), []byte("x"), 0o644); err != nil {
		t.Fatalf(errPrepFile, err)
	}
	if err = FileSave([]byte("x"), tplFileOut, filepath.Join(dir, tplBlocker, "sub")); err == nil {
		t.Error(errNoStat)
	}
}

// FileSave возвращает ошибку, если директорию невозможно создать.
func TestFileSaveMkdirError(t *testing.T) {
	const (
		errMkdirFail string = "сбой mkdir"
		errNoMkdir   string = "ожидалась ошибка создания директории"
	)
	var (
		orig func(string, os.FileMode) error
		err  error
	)

	orig = mkdirAll
	mkdirAll = func(path string, perm os.FileMode) error { return errors.New(errMkdirFail) }
	defer func() { mkdirAll = orig }()
	if err = FileSave([]byte("x"), tplFileOut, filepath.Join(t.TempDir(), "sub")); err == nil {
		t.Error(errNoMkdir)
	}
}

// FileSave возвращает ошибку, если файл невозможно открыть.
func TestFileSaveOpenError(t *testing.T) {
	const errNoOpen string = "ожидалась ошибка открытия файла в несуществующей поддиректории"
	var err error

	if err = FileSave([]byte("x"), filepath.Join("нет", tplFileOut), t.TempDir()); err == nil {
		t.Error(errNoOpen)
	}
}
