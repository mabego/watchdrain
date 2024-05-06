`watchdrain` is a simple command line tool that monitors a directory until it is empty of files.

I wrote it as an exercise to experiment with Go channels, goroutines, and testing in Go.

## How to Install

### Using `go run`

```shell
git clone https://github.com/mabego/watchdrain.git
```

```shell
cd watchdrain
```

```shell
go run . <dir>
```

### Building a binary


```shell
git clone https://github.com/mabego/watchdrain.git
```

```shell
cd watchdrain
```

```shell
go install
```

```shell
watchdrain <dir>
```

### CLI Options

```shell
watchdrain -h
watchdrain watches a directory until it is empty of files
Usage of watchdrain:
watchdrain <dir>
  -threshold uint
        Stop watching a directory if file create events exceed remove events by a threshold
        threshold = create events - remove events
        The lowest threshold is 1. Increase to allow more create events while watching.
  -timer duration
        Set a timer. Default is 5 minutes. (default 5m0s)
  -v    Log file create and remove events
watchdrain -timer 1m -threshold 1 -v <dir>
```
