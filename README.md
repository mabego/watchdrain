`watchdrain` is a simple command line tool that monitors a directory until it is empty of files.

I wrote it as an exercise to experiment with Go channels, goroutines, and testing in Go.

## Installation

```shell
go install github.com/mabego/watchdrain@latest
```

## Using `go run`

```shell
git clone https://github.com/mabego/watchdrain.git
```

```shell
cd watchdrain
```

```shell
go run . <directory>
```

## Usage

```shell
watchdrain <directory>
```

See `watchdrain --help` for more information.
