package puterin

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var _ io.WriteCloser = (*Writer)(nil)

type timeFunc = func() time.Time

var nowFunc timeFunc = time.Now

type Writer struct {
	Filename  string
	DirPath   string
	MaxAge    int
	file      *os.File
	mu        sync.Mutex
	lastWrite time.Time
}

type logInfo struct {
	timestamp time.Time
	os.FileInfo
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

	if w.isDateBefore(w.lastWrite, nowFunc()) {
		err := w.rotate()
		if err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	if err != nil {
		return 0, fmt.Errorf("failed to write log: %w", err)
	}

	w.lastWrite = nowFunc()
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
	return w.rotate()
}

func (w *Writer) openNewOrExisting() error {
	err := os.MkdirAll(w.DirPath, 0755)
	if err != nil {
		return fmt.Errorf("cannot create log dir: %w", err)
	}

	fileInfo, err := os.Stat(w.pathToFile())
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
	w.lastWrite = fileInfo.ModTime()

	return nil
}

func (w *Writer) openNew() error {
	file, err := os.OpenFile(w.pathToFile(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error opening new log file: %w", err)
	}

	w.file = file
	w.lastWrite = nowFunc()
	return nil
}

func (w *Writer) makeBackup() error {
	err := w.close()
	if err != nil {
		return err
	}

	ext := filepath.Ext(w.Filename)
	rawFilename := w.Filename[:len(w.Filename)-len(ext)]
	timeSuffix := w.lastWrite.Format("02-01-2006T15-04-05")
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

func (w *Writer) rotate() error {
	err := w.makeBackup()
	if err != nil {
		return fmt.Errorf("error creating backup: %w", err)
	}

	if w.MaxAge > 0 {
		go w.cleanExpiredLogs()
	}

	return w.openNew()
}

func (w *Writer) cleanExpiredLogs() {
	oldLogs, _ := w.oldLogFiles()
	thresholdDate := nowFunc().AddDate(0, 0, -w.MaxAge)
	for _, val := range oldLogs {
		if w.isDateBefore(val.timestamp, thresholdDate) {
			_ = os.Remove(filepath.Join(w.DirPath, val.Name()))
		}
	}
}

func (w *Writer) oldLogFiles() ([]logInfo, error) {
	files, err := ioutil.ReadDir(w.DirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log dir: %w", err)
	}

	ext := filepath.Ext(w.Filename)
	filename := w.Filename[:len(w.Filename)-len(ext)]

	var oldLogs []logInfo
	for _, val := range files {
		if t, err := time.Parse(fmt.Sprintf("%s-02-01-2006T15-04-05%s", filename, ext), val.Name()); err == nil {
			oldLogs = append(oldLogs, logInfo{timestamp: t, FileInfo: val})
		}
	}

	return oldLogs, nil
}

func (w *Writer) pathToFile() string {
	return filepath.Join(w.DirPath, w.Filename)
}

func (w *Writer) isDateBefore(n1 time.Time, n2 time.Time) bool {
	if n1.Year() != n2.Year() {
		return n1.Year() < n2.Year()
	}

	return n1.YearDay() < n2.YearDay()
}
