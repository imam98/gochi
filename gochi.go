package gochi

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var _ io.WriteCloser = (*Writer)(nil)

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

	if w.file == nil {
		return nil
	}

	err := w.file.Sync()
	_ = w.file.Close()
	w.file = nil
	return err
}

func (w *Writer) openNewOrExisting() error {
	err := os.MkdirAll(w.DirPath, 0755)
	if err != nil {
		return fmt.Errorf("cannot create log dir: %w", err)
	}

	mode := os.FileMode(0600)
	_, err = os.Stat(w.pathToFile())
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.OpenFile(w.pathToFile(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("error opening new log file: %w", err)
			}

			w.file = file
			return nil
		}

		return fmt.Errorf("unexpected error: %w", err)
	}

	file, err := os.OpenFile(w.pathToFile(), os.O_APPEND|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	w.file = file

	return nil
}

func (w *Writer) pathToFile() string {
	return filepath.Join(w.DirPath, w.Filename)
}
