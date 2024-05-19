package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/goleak"
)

const (
	testDir = "test-dir"
	sub     = "test-sub"
	file1   = "temp1.txt"
	file2   = "temp2.txt"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)

	fmt.Println("Running tests...")
	testsResult := m.Run()
	os.Exit(testsResult)
}

func createPath(t *testing.T) string {
	t.Helper()
	fmt.Println("Creating test directory path...")
	// Creating a testing directory and subdirectory in the default directory for temporary files
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, testDir, sub), 0o700); err != nil {
		t.Fatal(err)
	}
	tmpDir = filepath.Join(tmpDir, testDir)
	fmt.Printf("%s\n", tmpDir)

	return tmpDir
}

func createSeedFiles(t *testing.T, tmpDir string) {
	t.Helper()
	fmt.Println("Creating testing files...")
	files := []string{file1, file2}
	for _, file := range files {
		f, err := os.Create(filepath.Join(tmpDir, file))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Sync(); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%s\n", f.Name())
		if _, err := f.WriteString("ready to drain"); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
		if err := f.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func createTempFile(t *testing.T, tmpDir string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(tmpDir, "temp.*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if _, err := f.WriteString("ready to drain"); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReadDirFiles(t *testing.T) {
	testPath := createPath(t)

	for i := 0; i < 3; i++ {
		createTempFile(t, testPath)
	}

	d, err := newDir(testPath)
	if err != nil {
		t.Fatal(err)
	}

	want := uint32(3)
	got := *d.files
	if got != want {
		t.Errorf("Did not get expected result. Wanted: %d, got: %d", want, got)
	}
}

func TestEmpty(t *testing.T) {
	testPath := createPath(t)

	d, err := newDir(testPath)
	if err != nil {
		t.Fatal(err)
	}

	want := uint32(0)
	got := *d.files
	if got != want {
		t.Errorf("Did not get expected result. Wanted: %d, got: %d", want, got)
	}
}

func TestDeadline(t *testing.T) {
	testPath := createPath(t)
	createSeedFiles(t, testPath)

	want := ErrTimeout
	d, err := newDir(testPath)
	if err != nil {
		t.Fatal(err)
	}
	opts := newOptions((50 * time.Millisecond), 0, false)
	if _, got := d.watchDrain(opts); got != nil {
		if !errors.Is(got, want) {
			t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
		}
	}
}

func TestDrainNoCreates(t *testing.T) {
	testPath := createPath(t)
	createSeedFiles(t, testPath)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		d, err := newDir(testPath)
		if err != nil {
			t.Fatal(err)
		}
		opts := newOptions(0, 0, false)
		got, err := d.watchDrain(opts)
		if err != nil {
			t.Fatal(err)
		}
		if got != true {
			t.Errorf("Unexpected result. Wanted: %t, got: %t", true, got)
		}
	})

	t.Run("Drain", func(t *testing.T) {
		t.Parallel()

		time.Sleep(50 * time.Millisecond)
		if err := os.Remove(filepath.Join(testPath, file1)); err != nil {
			t.Error(err)
		}

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(testPath, file2)); err != nil {
			t.Error(err)
		}
	})
}

func TestDrainWithCreates(t *testing.T) {
	testPath := createPath(t)
	createSeedFiles(t, testPath)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		d, err := newDir(testPath)
		if err != nil {
			t.Fatal(err)
		}
		opts := newOptions((1 * time.Minute), math.MaxUint32, true)
		got, err := d.watchDrain(opts)
		if err != nil {
			t.Fatal(err)
		}
		if got != true {
			t.Errorf("Unexpected result. Wanted: %t, got: %t", true, got)
		}
	})

	t.Run("Drain", func(t *testing.T) {
		t.Parallel()

		time.Sleep(50 * time.Millisecond)
		if err := os.Remove(filepath.Join(testPath, file1)); err != nil {
			t.Error(err)
		}

		f3 := createTempFile(t, testPath)

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(testPath, file2)); err != nil {
			t.Error(err)
		}

		f4 := createTempFile(t, testPath)

		time.Sleep(time.Millisecond)
		if err := os.Remove(f3.Name()); err != nil {
			t.Error(err)
		}
		time.Sleep(time.Millisecond)
		if err := os.Remove(f4.Name()); err != nil {
			t.Error(err)
		}
	})
}

func TestNotDraining(t *testing.T) {
	testPath := createPath(t)
	createSeedFiles(t, testPath)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		want := ErrTooManyCreateEvents
		d, err := newDir(testPath)
		if err != nil {
			t.Fatal(err)
		}
		opts := newOptions((1 * time.Minute), 1, true)
		if _, got := d.watchDrain(opts); got != nil {
			if !errors.Is(got, want) {
				t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
			}
		}
	})

	t.Run("Drain", func(t *testing.T) {
		t.Parallel()

		time.Sleep(50 * time.Millisecond)
		createTempFile(t, testPath)

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(testPath, file1)); err != nil {
			t.Error(err)
		}

		createTempFile(t, testPath)

		// Exceed the create-to-remove threshold
		createTempFile(t, testPath)
	})
}
