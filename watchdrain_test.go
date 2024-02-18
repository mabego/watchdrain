package watchdrain

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
	testDir = "test-Dir"
	sub     = "test-sub"
	file1   = "temp1.txt"
	file2   = "temp2.txt"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
	fmt.Println("Cleaning path...")
	if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Running test...")
	result := m.Run()
	os.Exit(result)
}

func createPath(t *testing.T) {
	t.Helper()
	fmt.Println("Creating testing directory...")
	// Creating testing directory and subdirectory in the default directory for temporary files
	if err := os.MkdirAll(filepath.Join(os.TempDir(), testDir, sub), 0o700); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", filepath.Join(os.TempDir(), testDir))
}

func createSeedFiles(t *testing.T) {
	t.Helper()
	fmt.Println("Creating testing files...")
	files := []string{file1, file2}
	for _, file := range files {
		f, err := os.Create(filepath.Join(os.TempDir(), testDir, file))
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

func createTempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(filepath.Join(os.TempDir(), testDir), "temp.*.txt")
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
	createPath(t)

	for i := 0; i < 3; i++ {
		createTempFile(t)
	}

	d, err := NewDir(filepath.Join(os.TempDir(), testDir))
	if err != nil {
		t.Fatal(err)
	}

	want := uint32(3)
	got := *d.files
	if got != want {
		t.Errorf("Did not get expected result. Wanted: %d, got: %d", want, got)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEmpty(t *testing.T) {
	createPath(t)

	d, err := NewDir(filepath.Join(os.TempDir(), testDir))
	if err != nil {
		t.Fatal(err)
	}

	want := uint32(0)
	got := *d.files
	if got != want {
		t.Errorf("Did not get expected result. Wanted: %d, got: %d", want, got)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTimer(t *testing.T) {
	createPath(t)
	createSeedFiles(t)

	want := ErrTimerEnded
	d, err := NewDir(filepath.Join(os.TempDir(), testDir))
	if err != nil {
		t.Fatal(err)
	}
	opts := NewOptions((50 * time.Millisecond), 0, false)
	if _, got := d.WatchDrain(opts); got != nil {
		if !errors.Is(got, want) {
			t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
		}
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainNoCreates(t *testing.T) {
	createPath(t)
	createSeedFiles(t)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		d, err := NewDir(filepath.Join(os.TempDir(), testDir))
		if err != nil {
			t.Fatal(err)
		}
		opts := NewOptions(0, 0, false)
		got, err := d.WatchDrain(opts)
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
		if err := os.Remove(filepath.Join(os.TempDir(), testDir, file1)); err != nil {
			t.Error(err)
		}

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(os.TempDir(), testDir, file2)); err != nil {
			t.Error(err)
		}
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainWithCreates(t *testing.T) {
	createPath(t)
	createSeedFiles(t)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		d, err := NewDir(filepath.Join(os.TempDir(), testDir))
		if err != nil {
			t.Fatal(err)
		}
		opts := NewOptions((1 * time.Minute), math.MaxUint32, true)
		got, err := d.WatchDrain(opts)
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
		if err := os.Remove(filepath.Join(os.TempDir(), testDir, file1)); err != nil {
			t.Error(err)
		}

		f3 := createTempFile(t)

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(os.TempDir(), testDir, file2)); err != nil {
			t.Error(err)
		}

		f4 := createTempFile(t)

		time.Sleep(time.Millisecond)
		if err := os.Remove(f3.Name()); err != nil {
			t.Error(err)
		}
		time.Sleep(time.Millisecond)
		if err := os.Remove(f4.Name()); err != nil {
			t.Error(err)
		}
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestNotDraining(t *testing.T) {
	createPath(t)
	createSeedFiles(t)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()

		want := ErrThresholdExceeded
		d, err := NewDir(filepath.Join(os.TempDir(), testDir))
		if err != nil {
			t.Fatal(err)
		}
		opts := NewOptions((1 * time.Minute), 1, true)
		if _, got := d.WatchDrain(opts); got != nil {
			if !errors.Is(got, want) {
				t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
			}
		}
	})

	t.Run("Drain", func(t *testing.T) {
		t.Parallel()

		time.Sleep(50 * time.Millisecond)
		createTempFile(t)

		time.Sleep(time.Millisecond)
		if err := os.Remove(filepath.Join(os.TempDir(), testDir, file1)); err != nil {
			t.Error(err)
		}

		createTempFile(t)

		// Exceed the create-to-remove threshold
		createTempFile(t)
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}
