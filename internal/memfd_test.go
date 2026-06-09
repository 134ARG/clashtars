package internal

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestCreateMemfdExecutable(t *testing.T) {
	fd, path, err := CreateMemfdExecutable("clashtars-test", []byte("#!/bin/sh\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("memfd path is not visible: %v", err)
	}
}
