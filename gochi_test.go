package gochi

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	logdir, err := makeTempDir("TestWrite")
	require.NoError(t, err)
	defer os.RemoveAll(logdir)

	data := []byte("foooo")

	t.Run("file not exists", func(t *testing.T) {
		gochiWriter := &Writer{
			Filename: "test_write.log",
			DirPath:  logdir,
		}
		defer gochiWriter.Close()

		n, err := gochiWriter.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
	})

	t.Run("file exists", func(t *testing.T) {
		gochiWriter := &Writer{
			Filename: "test_write.log",
			DirPath:  logdir,
		}
		defer gochiWriter.Close()

		newData := []byte("baaaaarr")
		data = append(data, newData...)
		n, err := gochiWriter.Write(newData)
		require.NoError(t, err)
		assert.Equal(t, len(newData), n)
		assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
	})
}

func TestMakeLogDir(t *testing.T) {
	logdir := filepath.Join(os.TempDir(), "TestDir")
	defer os.RemoveAll(logdir)

	gochiWriter := &Writer{
		Filename: "test_log.log",
		DirPath:  logdir,
	}
	defer gochiWriter.Close()

	data := []byte("foooo")
	n, err := gochiWriter.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	fileInfo, err := os.Stat(filepath.Join(gochiWriter.DirPath, gochiWriter.Filename))
	require.NoError(t, err)
	assert.EqualValues(t, len(data), fileInfo.Size())
	assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
	assertFileCount(t, gochiWriter.DirPath, 1)
}

func assertContentMatch(t *testing.T, logFile string, value []byte) {
	fileInfo, err := os.Stat(logFile)
	require.NoError(t, err)
	assert.EqualValues(t, len(value), fileInfo.Size())

	buf, err := ioutil.ReadFile(logFile)
	require.NoError(t, err)
	assert.Equal(t, value, buf)
}

func assertFileCount(t *testing.T, dirpath string, expected int) {
	files, err := ioutil.ReadDir(dirpath)
	require.NoError(t, err)
	assert.EqualValues(t, expected, len(files))
}

func makeTempDir(name string) (string, error) {
	dir := filepath.Join(os.TempDir(), name)
	err := os.Mkdir(dir, 0700)
	return dir, err
}
