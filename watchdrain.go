package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// dir represents a directory to watch drain of files
type dir struct {
	mu      sync.RWMutex // mu guards files
	dirName *string
	files   *uint32
}

// newDir returns a new dir to watch drain
func newDir(dirName string) (*dir, error) {
	files, err := readDirFiles(dirName)
	if err != nil {
		return nil, err
	}
	dir := &dir{
		dirName: &dirName,
		files:   files,
	}
	return dir, nil
}

// readDirFiles reads a directory and returns a file count, ignoring subdirectories
func readDirFiles(dirName string) (*uint32, error) {
	d, err := os.Open(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory: %w", err)
	}
	defer d.Close()
	entries, err := d.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to get file count: %w", err)
	}
	var f uint32
	for _, entry := range entries {
		if !entry.IsDir() {
			f++
		}
	}
	return &f, nil
}

func (d *dir) isEmpty() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	f := *d.files
	return f == 0
}

var (
	// ErrThresholdExceeded is returned when file creation events exceed removal events by a set threshold
	ErrThresholdExceeded = errors.New("threshold exceeded")
	ErrTimerEnded        = errors.New("timer ended")
)

// event describes a set of file operation notifications
type event uint8

// Events that trigger a notification to eventCh
const (
	Create event = iota
	Remove
)

// options for watchDrain
type options struct {
	eventCh   chan event
	timer     time.Duration
	threshold uint
	verbose   bool
}

// newOptions returns options, including an eventCh channel if a threshold is set
func newOptions(timer time.Duration, threshold uint, verbose bool) *options {
	opts := &options{
		timer:     timer,
		threshold: threshold,
		verbose:   verbose,
	}
	if threshold > 0 {
		opts.eventCh = make(chan event)
	}
	return opts
}

// result provides return values for watchDrain
type result struct {
	drained bool
	err     error
}

// watchDrain watches a directory until it is empty of files or a timer ends or a threshold is exceeded
func (d *dir) watchDrain(opt *options) (bool, error) {
	ctx := context.Background()
	draining, cancel := context.WithCancel(ctx)
	resultCh := make(chan result)
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatalln(err)
	}
	if err := watcher.Add(*d.dirName); err != nil {
		log.Fatalln(err)
	}

	defer func() {
		cancel()
		close(resultCh)
		if err := watcher.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// Start watching the directory drain
	go drainer(d, watcher, draining, resultCh, opt)

	// Start the timer and/or monitoring threshold
	if opt.timer > 0 {
		go timer(ctx, draining, resultCh, opt)
	}
	if opt.threshold > 0 {
		go threshold(draining, resultCh, opt)
	}

	res := <-resultCh
	if res.err != nil {
		return false, res.err
	}
	return res.drained, nil
}

// drainer runs while the target directory is not empty, tracking file deletion and creation events
func drainer(d *dir, watcher *fsnotify.Watcher, draining context.Context, resultCh chan<- result, opt *options) {
	defer func() {
		if opt.threshold > 0 {
			close(opt.eventCh)
		}
	}()
	for !d.isEmpty() {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				if opt.verbose {
					log.Printf("%s EVENT: %s\n", event.Op, event.Name)
				}
				d.mu.Lock()
				*d.files--
				d.mu.Unlock()
				if opt.threshold > 0 {
					opt.eventCh <- Remove
				}
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				if opt.verbose {
					log.Printf("%s EVENT: %s\n", event.Op, event.Name)
				}
				d.mu.Lock()
				*d.files++
				d.mu.Unlock()
				if opt.threshold > 0 {
					opt.eventCh <- Create
				}
			}
		case err, ok := <-watcher.Errors:
			if ok {
				resultCh <- result{err: err}
				<-draining.Done()
			}
		}
	}
	resultCh <- result{drained: true}
	<-draining.Done()
}

func timer(ctx, draining context.Context, resultCh chan<- result, opt *options) {
	timer, cancelTimer := context.WithTimeout(ctx, opt.timer)
	defer cancelTimer()

	select {
	case <-timer.Done():
		resultCh <- result{err: ErrTimerEnded}
		<-draining.Done()
	case <-draining.Done():
		return
	}
}

// threshold monitors file creation activity.
// If file creation is too active and the directory is not going to drain, watchdrain will stop.
func threshold(draining context.Context, resultCh chan<- result, opt *options) {
	// `creates` and `removes` track draining activity
	creates, removes := 0, 0
	for {
		if event, ok := <-opt.eventCh; true {
			if !ok {
				return
			}
			switch {
			case event == Remove:
				removes++
			case event == Create:
				creates++
			}
		}
		if creates-removes > int(opt.threshold) { // 1 is the lowest threshold
			resultCh <- result{err: ErrThresholdExceeded}
			<-draining.Done()
			return
		}
	}
}
