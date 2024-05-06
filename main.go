package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	timer := flag.Duration("timer", (5 * time.Minute), "Set a timer. Default is 5 minutes.")
	threshold := flag.Uint("threshold", 0, "Stop watching a directory if file create events exceed "+
		"remove events by a threshold\nthreshold = create events - remove events\nThe lowest threshold is 1. "+
		"Increase to allow more create events while watching.")
	verbose := flag.Bool("v", false, "Log file create and remove events")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "watchdrain watches a directory until it is empty of files\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of watchdrain:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "%s <dir>\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "%s -timer 1m -threshold 1 -v <dir>\n", os.Args[0])
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
		opts := newOptions(*timer, *threshold, *verbose)
		watch, err := d.watchDrain(opts)
		if errors.Is(err, ErrTimerEnded) {
			fmt.Fprintf(os.Stderr, "%s: %s after %s\n", dir, err, timer)
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
