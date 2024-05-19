package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	deadline := flag.Duration("deadline", (5 * time.Minute), "Set a time to stop watching a directory "+
		"draining of files.")
	eventMonitor := flag.Uint("eventMonitor", 0, "Set a file creation monitor threshold to stop"+
		" watching a directory when file create events exceed remove events by a threshold:"+
		"\nthreshold = create events - remove events\n"+
		"Increase to allow more file creation activity while watching. The lowest threshold is 1.")
	verbose := flag.Bool("v", false, "Log file create and remove events")

	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage:\n %s [options] <dir>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	switch {
	case len(flag.Args()) == 1:
		dir := flag.Arg(0)
		d, err := newDir(dir)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		opts := newOptions(*deadline, *eventMonitor, *verbose)
		watch, err := d.watchDrain(opts)
		if errors.Is(err, ErrTimeout) {
			fmt.Fprintf(os.Stderr, "%s: %s after %s\n", dir, err, deadline)
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", dir, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "%s drained:%t\n", dir, watch)
		os.Exit(0)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
