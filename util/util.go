package util

import (
	"log/slog"
	"os"
	"runtime"
	"strconv"
)

func ErrExit(err error) {
	_, file, line, _ := runtime.Caller(1)
	slog.Error("A fatal error has occurred", "err", err, "caller", file+":"+strconv.Itoa(line))
	os.Exit(1)
}
