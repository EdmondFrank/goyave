package osfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"goyave.dev/goyave/v5/util/fsutil"
)

func setRootWorkingDirectory() {
	sep := string(os.PathSeparator)
	_, filename, _, _ := runtime.Caller(1)
	directory := path.Dir(filename) + sep
	for !fsutil.FileExists(&FS{}, directory+sep+"go.mod") {
		directory += ".." + sep
		if !fsutil.IsDirectory(&FS{}, directory) {
			panic("Couldn't find project's root directory.")
		}
	}
	if err := os.Chdir(directory); err != nil {
		panic(err)
	}
}

func TestOSFS(t *testing.T) {

	setRootWorkingDirectory()

	t.Run("Open", func(t *testing.T) {
		fs := &FS{}
		file, err := fs.Open("resources/test_file.txt")
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, file.Close())
		}()
		contents, err := io.ReadAll(file)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, append([]byte{0xef, 0xbb, 0xbf}, []byte("utf-8 with BOM content")...), contents)
	})

	t.Run("OpenFile", func(t *testing.T) {
		fs := &FS{}
		file, err := fs.OpenFile("resources/test_file.txt", os.O_RDONLY, 0660)
		if !assert.NoError(t, err) {
			return
		}
		defer func() {
			assert.NoError(t, file.Close())
		}()
		contents, err := io.ReadAll(file)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, append([]byte{0xef, 0xbb, 0xbf}, []byte("utf-8 with BOM content")...), contents)
	})

	t.Run("ReadDir", func(t *testing.T) {
		osfs := &FS{}
		entries, err := osfs.ReadDir("resources/lang")
		if !assert.NoError(t, err) {
			return
		}

		type result struct {
			name  string
			isdir bool
		}

		expected := []result{
			{name: "en-UK", isdir: true},
			{name: "en-US", isdir: true},
			{name: "invalid.json", isdir: false},
		}

		assert.Equal(t, expected, lo.Map(entries, func(e fs.DirEntry, _ int) result {
			return result{name: e.Name(), isdir: e.IsDir()}
		}))
	})

	t.Run("Stat", func(t *testing.T) {
		fs := &FS{}
		info, err := fs.Stat("resources/test_file.txt")
		if !assert.NoError(t, err) {
			return
		}

		assert.False(t, info.IsDir())
		assert.Equal(t, "test_file.txt", info.Name())
	})

	t.Run("Getwd", func(t *testing.T) {
		fs := &FS{}
		wd, err := fs.Getwd()
		if !assert.NoError(t, err) {
			return
		}
		assert.NotEmpty(t, wd)
	})

	t.Run("FileExists", func(t *testing.T) {
		fs := &FS{}
		assert.True(t, fs.FileExists("resources/test_file.txt"))
		assert.False(t, fs.FileExists("resources"))
		assert.False(t, fs.FileExists("resources/notafile.txt"))
	})

	t.Run("IsDirectory", func(t *testing.T) {
		fs := &FS{}
		assert.False(t, fs.IsDirectory("resources/test_file.txt"))
		assert.True(t, fs.IsDirectory("resources"))
		assert.False(t, fs.IsDirectory("resources/notadir"))
	})

	t.Run("Mkdir", func(t *testing.T) {
		fs := &FS{}
		path := "resources/testdir"
		assert.NoError(t, fs.Mkdir(path, 0770))
		assert.True(t, fs.IsDirectory(path))
		assert.NoError(t, os.RemoveAll(path))
	})

	t.Run("MkdirAll", func(t *testing.T) {
		fs := &FS{}
		path := "resources/testdirall/subdir"
		assert.NoError(t, fs.MkdirAll(path, 0770))
		assert.True(t, fs.IsDirectory(path))
		assert.NoError(t, os.RemoveAll(path))
	})
}