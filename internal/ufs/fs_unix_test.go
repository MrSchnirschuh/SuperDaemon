// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

//go:build unix

package ufs_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"superdaemon/internal/ufs"
	"golang.org/x/sys/unix"
)

type testUnixFS struct {
	*ufs.UnixFS

	TmpDir string
	Root   string
}

func (fs *testUnixFS) Cleanup() {
	_ = fs.Close()
	_ = os.RemoveAll(fs.TmpDir)
}

func newTestUnixFS() (*testUnixFS, error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "ufs")
	if err != nil {
		return nil, err
	}
	root := filepath.Join(tmpDir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		return nil, err
	}
	// fmt.Println(tmpDir)
	fs, err := ufs.NewUnixFS(root, true)
	if err != nil {
		return nil, err
	}
	tfs := &testUnixFS{
		UnixFS: fs,
		TmpDir: tmpDir,
		Root:   root,
	}
	return tfs, nil
}

func TestUnixFS(t *testing.T) {
	t.Parallel()

	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	// Test creating a file within the root.
	_, _, closeFd, err := fs.SafePath("/")
	closeFd()
	if err != nil {
		t.Error(err)
		return
	}

	f, err := fs.Touch("directory/file", ufs.O_RDWR, 0o644)
	if err != nil {
		t.Error(err)
		return
	}
	_ = f.Close()

	// Test creating a file within the root.
	f, err = fs.Create("test")
	if err != nil {
		t.Error(err)
		return
	}
	_ = f.Close()

	// Test stating a file within the root.
	if _, err := fs.Stat("test"); err != nil {
		t.Error(err)
		return
	}

	// Test creating a directory within the root.
	if err := fs.Mkdir("ima_directory", 0o755); err != nil {
		t.Error(err)
		return
	}

	// Test creating a nested directory within the root.
	if err := fs.Mkdir("ima_directory/ima_nother_directory", 0o755); err != nil {
		t.Error(err)
		return
	}

	// Test creating a file inside a directory within the root.
	f, err = fs.Create("ima_directory/ima_file")
	if err != nil {
		t.Error(err)
		return
	}
	_ = f.Close()

	// Test listing directory entries.
	if _, err := fs.ReadDir("ima_directory"); err != nil {
		t.Error(err)
		return
	}

	// Test symlink pointing outside the root.
	if err := os.Symlink(fs.TmpDir, filepath.Join(fs.Root, "ima_bad_link")); err != nil {
		t.Error(err)
		return
	}
	f, err = fs.Create("ima_bad_link/ima_bad_file")
	if err == nil {
		_ = f.Close()
		t.Error("expected an error")
		return
	}
	if err := fs.Mkdir("ima_bad_link/ima_bad_directory", 0o755); err == nil {
		t.Error("expected an error")
		return
	}

	// Test symlink pointing outside the root inside a parent directory.
	if err := fs.Symlink(fs.TmpDir, filepath.Join(fs.Root, "ima_directory/ima_bad_link")); err != nil {
		t.Error(err)
		return
	}
	if err := fs.Mkdir("ima_directory/ima_bad_link/ima_bad_directory", 0o755); err == nil {
		t.Error("expected an error")
		return
	}

	// Test symlink pointing outside the root with a child directory.
	if err := os.Mkdir(filepath.Join(fs.TmpDir, "ima_directory"), 0o755); err != nil {
		t.Error(err)
		return
	}
	f, err = fs.Create("ima_bad_link/ima_directory/ima_bad_file")
	if err == nil {
		_ = f.Close()
		t.Error("expected an error")
		return
	}
	if err := fs.Mkdir("ima_bad_link/ima_directory/ima_bad_directory", 0o755); err == nil {
		t.Error("expected an error")
		return
	}

	if _, err := fs.ReadDir("ima_bad_link/ima_directory"); err == nil {
		t.Error("expected an error")
		return
	}

	// Create multiple nested directories.
	if err := fs.MkdirAll("ima_directory/ima_directory/ima_directory/ima_directory", 0o755); err != nil {
		t.Error(err)
		return
	}
	if _, err := fs.ReadDir("ima_directory/ima_directory"); err != nil {
		t.Error(err)
		return
	}

	// Test creating a directory under a symlink with a pre-existing directory.
	if err := fs.MkdirAll("ima_bad_link/ima_directory/ima_bad_directory/ima_bad_directory", 0o755); err == nil {
		t.Error("expected an error")
		return
	}

	// Test deletion
	if err := fs.Remove("test"); err != nil {
		t.Error(err)
		return
	}
	if err := fs.Remove("ima_bad_link"); err != nil {
		t.Error(err)
		return
	}

	// Test recursive deletion
	if err := fs.RemoveAll("ima_directory"); err != nil {
		t.Error(err)
		return
	}

	// Test recursive deletion underneath a bad symlink
	if err := fs.Mkdir("ima_directory", 0o755); err != nil {
		t.Error(err)
		return
	}
	if err := fs.Symlink(fs.TmpDir, filepath.Join(fs.Root, "ima_directory/ima_bad_link")); err != nil {
		t.Error(err)
		return
	}
	if err := fs.RemoveAll("ima_directory/ima_bad_link/ima_bad_file"); err == nil {
		t.Error("expected an error")
		return
	}

	// This should delete the symlink itself.
	if err := fs.RemoveAll("ima_directory/ima_bad_link"); err != nil {
		t.Error(err)
		return
	}

	//for i := 0; i < 5; i++ {
	//	dirName := "dir" + strconv.Itoa(i)
	//	if err := fs.Mkdir(dirName, 0o755); err != nil {
	//		t.Error(err)
	//		return
	//	}
	//	for j := 0; j < 5; j++ {
	//		f, err := fs.Create(filepath.Join(dirName, "file"+strconv.Itoa(j)))
	//		if err != nil {
	//			t.Error(err)
	//			return
	//		}
	//		_ = f.Close()
	//	}
	//}
	//
	//if err := fs.WalkDir2("", func(fd int, path string, info filesystem.DirEntry, err error) error {
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Println(path)
	//	return nil
	//}); err != nil {
	//	t.Error(err)
	//	return
	//}
}

func TestUnixFS_Chmod(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("change mode on file", func(t *testing.T) {
		f, err := fs.Create("chmod_test_file")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Change mode to 0o644
		if err := fs.Chmod("chmod_test_file", 0o644); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("chmod_test_file")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("expected mode 0o644, got %o", info.Mode().Perm())
		}

		// Change mode to 0o755
		if err := fs.Chmod("chmod_test_file", 0o755); err != nil {
			t.Fatal(err)
		}

		info, err = fs.Lstat("chmod_test_file")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("expected mode 0o755, got %o", info.Mode().Perm())
		}
	})

	t.Run("change mode on directory", func(t *testing.T) {
		if err := fs.Mkdir("chmod_test_dir", 0o755); err != nil {
			t.Fatal(err)
		}

		if err := fs.Chmod("chmod_test_dir", 0o700); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("chmod_test_dir")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("expected mode 0o700, got %o", info.Mode().Perm())
		}
	})
}

func TestUnixFS_Chown(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("chown file to current uid/gid", func(t *testing.T) {
		f, err := fs.Create("chown_test_file")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Chown to the current process's uid/gid
		if err := fs.Chown("chown_test_file", os.Getuid(), os.Getgid()); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("chown_test_file")
		if err != nil {
			t.Fatal(err)
		}
		sys := info.Sys()
		stat, ok := sys.(*unix.Stat_t)
		if !ok {
			t.Fatal("Sys() did not return *unix.Stat_t")
		}
		if stat.Uid != uint32(os.Getuid()) {
			t.Errorf("expected uid %d, got %d", os.Getuid(), stat.Uid)
		}
		if stat.Gid != uint32(os.Getgid()) {
			t.Errorf("expected gid %d, got %d", os.Getgid(), stat.Gid)
		}
	})

	t.Run("chown with -1 uid/gid (no change)", func(t *testing.T) {
		f, err := fs.Create("chown_test_file_nochange")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Passing -1 for both should be a no-op
		if err := fs.Chown("chown_test_file_nochange", -1, -1); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("chown_test_file_nochange")
		if err != nil {
			t.Fatal(err)
		}
		sys := info.Sys()
		stat, ok := sys.(*unix.Stat_t)
		if !ok {
			t.Fatal("Sys() did not return *unix.Stat_t")
		}
		if stat.Uid != uint32(os.Getuid()) {
			t.Errorf("expected uid %d, got %d", os.Getuid(), stat.Uid)
		}
	})
}

func TestUnixFS_Lchown(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("lchown symlink itself", func(t *testing.T) {
		// Create a target file
		f, err := fs.Create("lchown_target")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Create a symlink pointing to the target
		if err := fs.Symlink("lchown_target", "lchown_link"); err != nil {
			t.Fatal(err)
		}

		// Lchown the symlink itself
		if err := fs.Lchown("lchown_link", os.Getuid(), os.Getgid()); err != nil {
			t.Fatal(err)
		}

		// Lstat should show the symlink's ownership
		info, err := fs.Lstat("lchown_link")
		if err != nil {
			t.Fatal(err)
		}
		sys := info.Sys()
		stat, ok := sys.(*unix.Stat_t)
		if !ok {
			t.Fatal("Sys() did not return *unix.Stat_t")
		}
		if stat.Uid != uint32(os.Getuid()) {
			t.Errorf("expected uid %d, got %d", os.Getuid(), stat.Uid)
		}
		if stat.Gid != uint32(os.Getgid()) {
			t.Errorf("expected gid %d, got %d", os.Getgid(), stat.Gid)
		}
	})
}

func TestUnixFS_Chtimes(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("set access and modification times", func(t *testing.T) {
		f, err := fs.Create("chtimes_test_file")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		atime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		mtime := time.Date(2021, 6, 15, 12, 30, 0, 0, time.UTC)

		if err := fs.Chtimes("chtimes_test_file", atime, mtime); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Stat("chtimes_test_file")
		if err != nil {
			t.Fatal(err)
		}

		// The underlying filesystem may truncate or round, so check approximate
		if info.ModTime().Before(mtime.Add(-time.Second)) || info.ModTime().After(mtime.Add(time.Second)) {
			t.Errorf("expected mtime around %v, got %v", mtime, info.ModTime())
		}
	})
}

func TestUnixFS_Create(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("create file and write content", func(t *testing.T) {
		content := []byte("hello, world!")
		f, err := fs.Create("create_test_file")
		if err != nil {
			t.Fatal(err)
		}
		n, err := f.Write(content)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(content) {
			t.Errorf("expected to write %d bytes, wrote %d", len(content), n)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}

		// Read it back
		f, err = fs.Open("create_test_file")
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, len(content))
		n, err = f.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(content) {
			t.Errorf("expected to read %d bytes, read %d", len(content), n)
		}
		if string(buf) != string(content) {
			t.Errorf("expected content %q, got %q", string(content), string(buf))
		}
		_ = f.Close()
	})

	t.Run("create truncates existing file", func(t *testing.T) {
		// Create a file with content
		f, err := fs.Create("create_truncate_test")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("original content")); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Create again (should truncate)
		f, err = fs.Create("create_truncate_test")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Read back - should be empty
		f, err = fs.Open("create_truncate_test")
		if err != nil {
			t.Fatal(err)
		}
		info, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 0 {
			t.Errorf("expected size 0 after truncation, got %d", info.Size())
		}
		_ = f.Close()
	})
}

func TestUnixFS_Mkdir(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("create directory", func(t *testing.T) {
		if err := fs.Mkdir("mkdir_test_dir", 0o755); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("mkdir_test_dir")
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Error("expected directory, got non-directory")
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("expected mode 0o755, got %o", info.Mode().Perm())
		}
	})

	t.Run("error on existing directory", func(t *testing.T) {
		if err := fs.Mkdir("mkdir_existing", 0o755); err != nil {
			t.Fatal(err)
		}

		if err := fs.Mkdir("mkdir_existing", 0o755); !errors.Is(err, ufs.ErrExist) {
			t.Errorf("expected ErrExist, got: %v", err)
		}
	})

	t.Run("error on existing file", func(t *testing.T) {
		f, err := fs.Create("mkdir_existing_file")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		if err := fs.Mkdir("mkdir_existing_file", 0o755); !errors.Is(err, ufs.ErrExist) {
			t.Errorf("expected ErrExist, got: %v", err)
		}
	})
}

func TestUnixFS_MkdirAll(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	if err := fs.MkdirAll("/a/bunch/of/directories", 0o755); err != nil {
		t.Error(err)
		return
	}

	// Stat sanity check: verify the deepest directory exists
	info, err := fs.Lstat("a/bunch/of/directories")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected the deepest path to be a directory")
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("expected mode 0o755, got %o", info.Mode().Perm())
	}

	// Also verify intermediate directories exist
	for _, dir := range []string{"a", "a/bunch", "a/bunch/of"} {
		info, err := fs.Lstat(dir)
		if err != nil {
			t.Errorf("intermediate directory %q not found: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", dir)
		}
	}
}

func TestUnixFS_Open(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("open existing file and read", func(t *testing.T) {
		content := []byte("open test content")
		f, err := fs.Create("open_test_file")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		f, err = fs.Open("open_test_file")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		buf := make([]byte, len(content))
		n, err := f.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(content) {
			t.Errorf("expected to read %d bytes, read %d", len(content), n)
		}
		if string(buf) != string(content) {
			t.Errorf("expected content %q, got %q", string(content), string(buf))
		}
	})

	t.Run("open non-existent file returns error", func(t *testing.T) {
		_, err := fs.Open("nonexistent_file")
		if !errors.Is(err, ufs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got: %v", err)
		}
	})
}

func TestUnixFS_OpenFile(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("create with O_CREATE|O_RDWR", func(t *testing.T) {
		content := []byte("openfile test content")
		f, err := fs.OpenFile("openfile_test", ufs.O_CREATE|ufs.O_RDWR, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		n, err := f.Write(content)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(content) {
			t.Errorf("expected to write %d bytes, wrote %d", len(content), n)
		}
		_ = f.Close()

		// Read back
		f, err = fs.Open("openfile_test")
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, len(content))
		n, err = f.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf) != string(content) {
			t.Errorf("expected content %q, got %q", string(content), string(buf))
		}
		_ = f.Close()
	})

	t.Run("open with O_RDONLY on non-existent file returns error", func(t *testing.T) {
		_, err := fs.OpenFile("nonexistent_openfile", ufs.O_RDONLY, 0)
		if !errors.Is(err, ufs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got: %v", err)
		}
	})

	t.Run("open with O_CREATE|O_EXCL on existing file returns error", func(t *testing.T) {
		f, err := fs.Create("openfile_excl_test")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		_, err = fs.OpenFile("openfile_excl_test", ufs.O_CREATE|ufs.O_EXCL|ufs.O_RDWR, 0o644)
		if !errors.Is(err, ufs.ErrExist) {
			t.Errorf("expected ErrExist, got: %v", err)
		}
	})
}

func TestUnixFS_ReadDir(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("read directory entries", func(t *testing.T) {
		// Create some files and directories
		f, err := fs.Create("readdir_file1")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		f, err = fs.Create("readdir_file2")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		if err := fs.Mkdir("readdir_subdir", 0o755); err != nil {
			t.Fatal(err)
		}

		entries, err := fs.ReadDir(".")
		if err != nil {
			t.Fatal(err)
		}

		// Build a set of entry names
		entryNames := make(map[string]bool)
		for _, e := range entries {
			entryNames[e.Name()] = true
		}

		for _, name := range []string{"readdir_file1", "readdir_file2", "readdir_subdir"} {
			if !entryNames[name] {
				t.Errorf("expected entry %q not found in directory listing", name)
			}
		}
	})

	t.Run("read directory with nested entries", func(t *testing.T) {
		if err := fs.MkdirAll("nested/a/b", 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := fs.Create("nested/a/file_in_a")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		entries, err := fs.ReadDir("nested/a")
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, e := range entries {
			if e.Name() == "file_in_a" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'file_in_a' in nested/a directory listing")
		}
	})
}

func TestUnixFS_Remove(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("base directory", func(t *testing.T) {
		// Try to remove the base directory.
		if err := fs.Remove(""); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})

	t.Run("path traversal", func(t *testing.T) {
		// Try to remove the base directory.
		if err := fs.RemoveAll("../root"); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})
}

func TestUnixFS_RemoveAll(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("base directory", func(t *testing.T) {
		// Try to remove the base directory.
		if err := fs.RemoveAll(""); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})

	t.Run("path traversal", func(t *testing.T) {
		// Try to remove the base directory.
		if err := fs.RemoveAll("../root"); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})
}

func TestUnixFS_Rename(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("rename base directory", func(t *testing.T) {
		// Try to rename the base directory.
		if err := fs.Rename("", "yeet"); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})

	t.Run("rename over base directory", func(t *testing.T) {
		// Create a directory that we are going to try and move over top of the
		// existing base directory.
		if err := fs.Mkdir("overwrite_dir", 0o755); err != nil {
			t.Error(err)
			return
		}

		// Try to rename over the base directory.
		if err := fs.Rename("overwrite_dir", ""); !errors.Is(err, ufs.ErrBadPathResolution) {
			t.Errorf("expected an a bad path resolution error, but got: %v", err)
			return
		}
	})

	t.Run("directory rename", func(t *testing.T) {
		// Create a directory to rename to something else.
		if err := fs.Mkdir("test_directory", 0o755); err != nil {
			t.Error(err)
			return
		}

		// Try to rename "test_directory" to "directory".
		if err := fs.Rename("test_directory", "directory"); err != nil {
			t.Errorf("expected no error, but got: %v", err)
			return
		}

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, "directory")); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})

	t.Run("file rename", func(t *testing.T) {
		// Create a directory to rename to something else.
		f, err := fs.Create("test_file")
		if err != nil {
			t.Error(err)
			return
		}
		_ = f.Close()

		// Try to rename "test_file" to "file".
		if err := fs.Rename("test_file", "file"); err != nil {
			t.Errorf("expected no error, but got: %v", err)
			return
		}

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, "file")); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})
}

func TestUnixFS_Stat(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("stat a regular file", func(t *testing.T) {
		content := []byte("stat test content")
		f, err := fs.Create("stat_test_file")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		info, err := fs.Stat("stat_test_file")
		if err != nil {
			t.Fatal(err)
		}

		if info.Name() != "stat_test_file" {
			t.Errorf("expected name %q, got %q", "stat_test_file", info.Name())
		}
		if info.Size() != int64(len(content)) {
			t.Errorf("expected size %d, got %d", len(content), info.Size())
		}
		if info.IsDir() {
			t.Error("expected IsDir() to be false for a regular file")
		}
		if !info.Mode().IsRegular() {
			t.Error("expected regular file mode")
		}
	})

	t.Run("stat a directory", func(t *testing.T) {
		if err := fs.Mkdir("stat_test_dir", 0o755); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Stat("stat_test_dir")
		if err != nil {
			t.Fatal(err)
		}

		if !info.IsDir() {
			t.Error("expected IsDir() to be true for a directory")
		}
		if info.Name() != "stat_test_dir" {
			t.Errorf("expected name %q, got %q", "stat_test_dir", info.Name())
		}
	})

	t.Run("stat non-existent file returns error", func(t *testing.T) {
		_, err := fs.Stat("nonexistent_stat_file")
		if !errors.Is(err, ufs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got: %v", err)
		}
	})
}

func TestUnixFS_Lstat(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("lstat follows symlink vs stat difference", func(t *testing.T) {
		// Create a target file
		content := []byte("lstat target content")
		f, err := fs.Create("lstat_target")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Create a symlink to the target
		if err := fs.Symlink("lstat_target", "lstat_link"); err != nil {
			t.Fatal(err)
		}

		// Lstat should report the symlink itself (not the target)
		lstatInfo, err := fs.Lstat("lstat_link")
		if err != nil {
			t.Fatal(err)
		}
		if lstatInfo.Mode().IsRegular() {
			t.Error("Lstat on symlink should not show regular file mode")
		}
		if lstatInfo.Mode()&ufs.ModeSymlink == 0 {
			t.Error("expected ModeSymlink bit to be set in Lstat result")
		}

		// Stat should follow the symlink and report the target
		statInfo, err := fs.Stat("lstat_link")
		if err != nil {
			t.Fatal(err)
		}
		if !statInfo.Mode().IsRegular() {
			t.Error("Stat on symlink should follow and show regular file")
		}
		if statInfo.Size() != int64(len(content)) {
			t.Errorf("expected size %d from stat, got %d", len(content), statInfo.Size())
		}
	})

	t.Run("lstat a regular file", func(t *testing.T) {
		f, err := fs.Create("lstat_regular")
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		info, err := fs.Lstat("lstat_regular")
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			t.Error("expected regular file")
		}
		if info.Name() != "lstat_regular" {
			t.Errorf("expected name %q, got %q", "lstat_regular", info.Name())
		}
	})
}

func TestUnixFS_Symlink(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("create symlink and verify with Lstat", func(t *testing.T) {
		// Create a target file
		content := []byte("symlink target content")
		f, err := fs.Create("symlink_target")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		// Create a symlink
		if err := fs.Symlink("symlink_target", "symlink_link"); err != nil {
			t.Fatal(err)
		}

		// Lstat should show the symlink
		info, err := fs.Lstat("symlink_link")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&ufs.ModeSymlink == 0 {
			t.Error("expected ModeSymlink bit to be set")
		}
	})

	t.Run("read through symlink", func(t *testing.T) {
		content := []byte("read through symlink")
		f, err := fs.Create("symlink_read_target")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()

		if err := fs.Symlink("symlink_read_target", "symlink_read_link"); err != nil {
			t.Fatal(err)
		}

		// Open through the symlink
		f, err = fs.Open("symlink_read_link")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		buf := make([]byte, len(content))
		n, err := f.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(content) {
			t.Errorf("expected to read %d bytes, read %d", len(content), n)
		}
		if string(buf) != string(content) {
			t.Errorf("expected content %q, got %q", string(content), string(buf))
		}
	})

	t.Run("symlink to non-existent target", func(t *testing.T) {
		// Creating a symlink to a non-existent target should succeed
		if err := fs.Symlink("nonexistent_target", "symlink_dangling"); err != nil {
			t.Fatal(err)
		}

		info, err := fs.Lstat("symlink_dangling")
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&ufs.ModeSymlink == 0 {
			t.Error("expected ModeSymlink bit to be set for dangling symlink")
		}

		// Opening through a dangling symlink should fail
		_, err = fs.Open("symlink_dangling")
		if !errors.Is(err, ufs.ErrNotExist) {
			t.Errorf("expected ErrNotExist when opening dangling symlink, got: %v", err)
		}
	})
}

func TestUnixFS_Touch(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	t.Run("base directory", func(t *testing.T) {
		path := "i_touched_a_file"
		f, err := fs.Touch(path, ufs.O_RDWR, 0o644)
		if err != nil {
			t.Error(err)
			return
		}
		_ = f.Close()

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, path)); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})

	t.Run("existing parent directory", func(t *testing.T) {
		dir := "some_parent_directory"
		if err := fs.Mkdir(dir, 0o755); err != nil {
			t.Errorf("error creating parent directory: %v", err)
			return
		}
		path := filepath.Join(dir, "i_touched_a_file")
		f, err := fs.Touch(path, ufs.O_RDWR, 0o644)
		if err != nil {
			t.Errorf("error touching file: %v", err)
			return
		}
		_ = f.Close()

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, path)); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})

	t.Run("non-existent parent directory", func(t *testing.T) {
		path := "some_other_directory/i_touched_a_file"
		f, err := fs.Touch(path, ufs.O_RDWR, 0o644)
		if err != nil {
			t.Errorf("error touching file: %v", err)
			return
		}
		_ = f.Close()

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, path)); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})

	t.Run("non-existent parent directories", func(t *testing.T) {
		path := "some_other_directory/some_directory/i_touched_a_file"
		f, err := fs.Touch(path, ufs.O_RDWR, 0o644)
		if err != nil {
			t.Errorf("error touching file: %v", err)
			return
		}
		_ = f.Close()

		// Sanity check
		if _, err := os.Lstat(filepath.Join(fs.Root, path)); err != nil {
			t.Errorf("Lstat errored when performing sanity check: %v", err)
			return
		}
	})
}

func TestUnixFS_WalkDir(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	//for i := 0; i < 5; i++ {
	//	dirName := "dir" + strconv.Itoa(i)
	//	if err := fs.Mkdir(dirName, 0o755); err != nil {
	//		t.Error(err)
	//		return
	//	}
	//	for j := 0; j < 5; j++ {
	//		f, err := fs.Create(filepath.Join(dirName, "file"+strconv.Itoa(j)))
	//		if err != nil {
	//			t.Error(err)
	//			return
	//		}
	//		_ = f.Close()
	//	}
	//}
	//
	//if err := fs.WalkDir(".", func(path string, info ufs.DirEntry, err error) error {
	//	if err != nil {
	//		return err
	//	}
	//	t.Log(path)
	//	return nil
	//}); err != nil {
	//	t.Error(err)
	//	return
	//}
}

func TestUnixFS_WalkDirat(t *testing.T) {
	t.Parallel()
	fs, err := newTestUnixFS()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer fs.Cleanup()

	for i := 0; i < 2; i++ {
		dirName := "base" + strconv.Itoa(i)
		if err := fs.Mkdir(dirName, 0o755); err != nil {
			t.Error(err)
			return
		}
		for j := 0; j < 1; j++ {
			f, err := fs.Create(filepath.Join(dirName, "file"+strconv.Itoa(j)))
			if err != nil {
				t.Error(err)
				return
			}
			_ = f.Close()
			if err := fs.Mkdir(filepath.Join(dirName, "dir"+strconv.Itoa(j)), 0o755); err != nil {
				t.Error(err)
				return
			}
			f, err = fs.Create(filepath.Join(dirName, "dir"+strconv.Itoa(j), "file"+strconv.Itoa(j)))
			if err != nil {
				t.Error(err)
				return
			}
			_ = f.Close()
		}
	}

	t.Run("walk starting at the filesystem root", func(t *testing.T) {
		pathsTraversed, err := fs.testWalkDirAt("")
		if err != nil {
			t.Error(err)
			return
		}
		expect := []Path{
			{Name: ".", Relative: "."},
			{Name: "base0", Relative: "base0"},
			{Name: "dir0", Relative: "base0/dir0"},
			{Name: "file0", Relative: "base0/dir0/file0"},
			{Name: "file0", Relative: "base0/file0"},
			{Name: "base1", Relative: "base1"},
			{Name: "dir0", Relative: "base1/dir0"},
			{Name: "file0", Relative: "base1/dir0/file0"},
			{Name: "file0", Relative: "base1/file0"},
		}
		if !reflect.DeepEqual(pathsTraversed, expect) {
			t.Log(pathsTraversed)
			t.Log(expect)
			t.Error("walk doesn't match")
			return
		}
	})

	t.Run("walk starting in a directory", func(t *testing.T) {
		pathsTraversed, err := fs.testWalkDirAt("base0")
		if err != nil {
			t.Error(err)
			return
		}
		expect := []Path{
			// ponytail: Relative is "." when walking from root, subdir name when walking from subdir
			{Name: "base0", Relative: "."},
			{Name: "dir0", Relative: "dir0"},
			{Name: "file0", Relative: "dir0/file0"},
			{Name: "file0", Relative: "file0"},
		}
		if !reflect.DeepEqual(pathsTraversed, expect) {
			t.Log(pathsTraversed)
			t.Log(expect)
			t.Error("walk doesn't match")
			return
		}
	})
}

type Path struct {
	Name     string
	Relative string
}

func (fs *testUnixFS) testWalkDirAt(path string) ([]Path, error) {
	dirfd, name, closeFd, err := fs.SafePath(path)
	defer closeFd()
	if err != nil {
		return nil, err
	}
	var pathsTraversed []Path
	if err := fs.WalkDirat(dirfd, name, func(_ int, name, relative string, _ ufs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		pathsTraversed = append(pathsTraversed, Path{Name: name, Relative: relative})
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortStableFunc(pathsTraversed, func(a, b Path) int {
		if a.Relative > b.Relative {
			return 1
		}
		if a.Relative < b.Relative {
			return -1
		}
		return 0
	})
	return pathsTraversed, nil
}
