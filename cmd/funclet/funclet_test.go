package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSetupSignalHandler(t *testing.T) {
	_, finishCh := SetupSignalHandler()
	pid := os.Getpid()
	syscall.Kill(pid, syscall.SIGTERM)
	syscall.Kill(pid, syscall.SIGCHLD)
	time.Sleep(time.Second)
	close(finishCh)
}
