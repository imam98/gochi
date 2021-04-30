package gochi

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	logdir, err := makeTempDir("TestWrite")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(logdir)

	gochiWriter := &Writer{
		Filename: "test_write.log",
		DirPath:  logdir,
	}
	defer gochiWriter.Close()

	data := []byte("foooo")
	n, err := gochiWriter.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
}

func assertContentMatch(t *testing.T, logFile string, value []byte) {
	fileInfo, err := os.Stat(logFile)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.EqualValues(t, len(value), fileInfo.Size())
	buf, err := ioutil.ReadFile(logFile)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.Equal(t, value, buf)
}

func makeTempDir(name string) (string, error) {
	dir := filepath.Join(os.TempDir(), name)
	err := os.Mkdir(dir, 0700)
	return dir, err
}
