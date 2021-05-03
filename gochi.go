package gochi

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var _ io.WriteCloser = (*Writer)(nil)

type timeFunc = func() time.Time

var nowFunc timeFunc = time.Now

type Writer struct {
	Filename string
	DirPath  string
	MaxAge   int
	file     *os.File
	mu       sync.Mutex
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		err := w.openNewOrExisting()
		if err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	if err != nil {
		return 0, fmt.Errorf("failed to write log: %w", err)
	}

	return n, nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.close()
}

func (w *Writer) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.makeBackup()
	if err != nil {
		return fmt.Errorf("error creating backup: %w", err)
	}

	return w.openNew()
}

func (w *Writer) openNewOrExisting() error {
	err := os.MkdirAll(w.DirPath, 0755)
	if err != nil {
		return fmt.Errorf("cannot create log dir: %w", err)
	}

	_, err = os.Stat(w.pathToFile())
	if err != nil {
		if os.IsNotExist(err) {
			return w.openNew()
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	file, err := os.OpenFile(w.pathToFile(), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	w.file = file

	return nil
}

func (w *Writer) openNew() error {
	file, err := os.OpenFile(w.pathToFile(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening new log file: %w", err)
	}

	w.file = file
	return nil
}

func (w *Writer) makeBackup() error {
	err := w.close()
	if err != nil {
		return err
	}

	ext := filepath.Ext(w.Filename)
	rawFilename := w.Filename[:len(w.Filename)-len(ext)]
	timeSuffix := nowFunc().Format("02-Jan-2006T15-04-05")
	newFilename := fmt.Sprintf("%s-%s%s", rawFilename, timeSuffix, ext)
	oldPath := filepath.Join(w.DirPath, w.Filename)
	newPath := filepath.Join(w.DirPath, newFilename)
	return os.Rename(oldPath, newPath)
}

func (w *Writer) close() error {
	if w.file == nil {
		return nil
	}

	err := w.file.Sync()
	_ = w.file.Close()
	w.file = nil
	return err
}

func (w *Writer) pathToFile() string {
	return filepath.Join(w.DirPath, w.Filename)
}
