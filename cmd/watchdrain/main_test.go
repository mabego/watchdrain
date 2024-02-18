package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go.uber.org/goleak"
)

const (
	file1   = "temp1.txt"
	file2   = "temp2.txt"
	sub     = "test-sub"
	testDir = "test-dir"
)

var binName = "watchdrain"

func TestMain(m *testing.M) {
	fmt.Println("Cleaning path...")
	if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
		fmt.Println(err)
	}
	fmt.Println("Building tool...")
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	build := exec.Command("go", "build", "-o", binName)

	if err := build.Run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Cannot build tool %s: %s", binName, err)
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", binName)

	fmt.Println("Running test...")
	result := m.Run()

	fmt.Println("Cleaning up...")
	if err := os.Remove(binName); err != nil {
		fmt.Println(err)
	}

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
	fmt.Println("Seeding testing directory with files...")
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

func TestEmptyDir(t *testing.T) {
	defer goleak.VerifyNone(t)
	createPath(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(wd, binName)

	t.Run("Watch", func(t *testing.T) {
		want := fmt.Sprintf("%s drained:true\n", filepath.Join(os.TempDir(), testDir))
		out, err := exec.Command(cmdPath, "-timer", "1m", "-threshold", "1", filepath.Join(os.TempDir(), testDir)).Output()
		if err != nil {
			t.Fatal(err)
		}
		got := string(out)
		if got != want {
			t.Errorf("Did not get expected result. Wanted: %s, got: %s", want, got)
		}
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTimer(t *testing.T) {
	defer goleak.VerifyNone(t)
	createPath(t)
	createSeedFiles(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(wd, binName)

	t.Run("Watch", func(t *testing.T) {
		want := fmt.Sprintf("%s: timer ended after 50ms\n", filepath.Join(os.TempDir(), testDir))
		cmd := exec.Command(cmdPath, "-timer", "50ms", "-threshold", "1", filepath.Join(os.TempDir(), testDir))
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		_ = cmd.Run()
		got := stderr.String()
		if got != want {
			t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
		}
	})

	t.Cleanup(func() {
		if err := os.RemoveAll(filepath.Join(os.TempDir(), testDir)); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDrainNoCreates(t *testing.T) {
	defer goleak.VerifyNone(t)
	createPath(t)
	createSeedFiles(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(wd, binName)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()
		want := fmt.Sprintf("%s drained:true\n", filepath.Join(os.TempDir(), testDir))
		out, err := exec.Command(cmdPath, "-timer", "1m", "-threshold", "1", filepath.Join(os.TempDir(), testDir)).Output()
		if err != nil {
			t.Fatal(err)
		}
		got := string(out)
		if got != want {
			t.Errorf("Did not get expected result. Wanted: %s, got: %s", want, got)
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
	defer goleak.VerifyNone(t)
	createPath(t)
	createSeedFiles(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(wd, binName)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()
		want := fmt.Sprintf("%s drained:true\n", filepath.Join(os.TempDir(), testDir))
		out, err := exec.Command(cmdPath, "-timer", "1m", "-threshold", "1", filepath.Join(os.TempDir(), testDir)).Output()
		if err != nil {
			t.Fatal(err)
		}
		got := string(out)
		if got != want {
			t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
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
	defer goleak.VerifyNone(t)
	createPath(t)
	createSeedFiles(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(wd, binName)

	t.Run("Watch", func(t *testing.T) {
		t.Parallel()
		want := fmt.Sprintf("%s: threshold exceeded\n", filepath.Join(os.TempDir(), testDir))
		cmd := exec.Command(cmdPath, "-timer", "1m", "-threshold", "1", filepath.Join(os.TempDir(), testDir))
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		_ = cmd.Run()
		got := stderr.String()
		if got != want {
			t.Errorf("Unexpected result. Wanted: %s, got: %s", want, got)
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
