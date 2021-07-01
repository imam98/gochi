package gochi

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
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

	t.Run("file exists different day", func(t *testing.T) {
		gochiWriter := &Writer{
			Filename: "test_write.log",
			DirPath:  logdir,
		}
		defer gochiWriter.Close()
		nowFunc = func() time.Time {
			return time.Now().AddDate(0, 0, 1)
		}

		n, err := gochiWriter.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
		assertFileCount(t, gochiWriter.DirPath, 2)
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

func TestRotate(t *testing.T) {
	logdir, err := makeTempDir("TestRotate")
	require.NoError(t, err)
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
	assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)

	err = gochiWriter.Rotate()
	require.NoError(t, err)
	assertFileCount(t, gochiWriter.DirPath, 2)
}

func TestWriteDifferentTime(t *testing.T) {
	testcases := []struct {
		name       string
		timeBefore string
		timeAfter  string
	}{
		{
			name:       "full 24 hours",
			timeBefore: "03-May-2021 13:00:00",
			timeAfter:  "04-May-2021 13:00:00",
		}, {
			name:       "hours to next day",
			timeBefore: "03-May-2021 23:00:00",
			timeAfter:  "04-May-2021 01:15:00",
		}, {
			name:       "minutes to next day",
			timeBefore: "03-May-2021 23:58:00",
			timeAfter:  "04-May-2021 00:01:00",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			logdir, err := makeTempDir("TestDifferentTime")
			require.NoError(t, err)
			defer os.RemoveAll(logdir)

			gochiWriter := &Writer{
				Filename: "test_log.log",
				DirPath:  logdir,
			}
			defer gochiWriter.Close()

			nowFunc = func() time.Time {
				val, _ := time.Parse("02-Jan-2006 15:04:05", tc.timeBefore)
				return val
			}

			data := []byte("foooo")
			n, err := gochiWriter.Write(data)
			require.NoError(t, err)
			assert.Equal(t, len(data), n)
			assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)

			nowFunc = func() time.Time {
				val, _ := time.Parse("02-Jan-2006 15:04:05", tc.timeAfter)
				return val
			}

			n, err = gochiWriter.Write(data)
			require.NoError(t, err)
			assert.Equal(t, len(data), n)
			assertContentMatch(t, filepath.Join(gochiWriter.DirPath, gochiWriter.Filename), data)
			assertFileCount(t, gochiWriter.DirPath, 2)
		})
	}
}

func TestGetOldLogFiles(t *testing.T) {
	logdir, err := makeTempDir("TestOldLogs")
	require.NoError(t, err)
	defer os.RemoveAll(logdir)

	gochiWriter := &Writer{
		Filename: "test_log.log",
		DirPath:  logdir,
	}
	defer gochiWriter.Close()

	mockTime, _ := time.Parse("02-Jan-2006 15:04:05", "06-May-2021 13:20:00")

	// The magic of closure
	var i int
	nowFunc = func() time.Time {
		return mockTime.AddDate(0, 0, i)
	}

	data := []byte("foooo")
	for i = 0; i <= 2; i++ {
		_, err := gochiWriter.Write(data)
		require.NoError(t, err)
	}
	assertFileCount(t, gochiWriter.DirPath, 3)

	files, err := gochiWriter.oldLogFiles()
	require.NoError(t, err)
	assert.Equal(t, 2, len(files))

	for _, val := range files {
		if !val.timestamp.Equal(mockTime) && !val.timestamp.Equal(mockTime.AddDate(0, 0, 1)) {
			t.Errorf("Unexpected log timestamp: %v", val.timestamp)
		}
	}
}

func TestRotateCleanExpiredLogs(t *testing.T) {
	logdir, err := makeTempDir("TestExpLogs")
	require.NoError(t, err)
	defer os.RemoveAll(logdir)

	gochiWriter := &Writer{
		Filename: "test_log.log",
		DirPath:  logdir,
		MaxAge:   1,
	}
	defer gochiWriter.Close()

	mockTime, _ := time.Parse("02-Jan-2006 15:04:05", "30-Jun-2021 16:00:00")

	// The magic of closure
	var i int
	nowFunc = func() time.Time {
		return mockTime.AddDate(0, 0, i)
	}

	data := []byte("foooo")
	for i = 0; i <= 2; i++ {
		_, err := gochiWriter.Write(data)
		require.NoError(t, err)
		// Delay for 100 ms so the writer can rotate peacefully
		time.Sleep(100 * time.Millisecond)
	}
	assertFileCount(t, gochiWriter.DirPath, 2)

	// Skip i = 3 and write at i = 4 (4 days later after mockTime)
	i = 4
	_, err = gochiWriter.Write(data)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
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

// TODO: Check behavior when dir path points to a file
