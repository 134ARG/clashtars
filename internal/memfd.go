package internal

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func CreateMemfdExecutable(name string, data []byte) (int, string, error) {
	if len(data) == 0 {
		return -1, "", fmt.Errorf("executable image is empty")
	}

	fd, err := unix.MemfdCreate(name, unix.MFD_CLOEXEC)
	if err != nil {
		return -1, "", err
	}

	if err := writeAll(fd, data); err != nil {
		_ = unix.Close(fd)
		return -1, "", err
	}
	if err := unix.Fchmod(fd, 0755); err != nil {
		_ = unix.Close(fd)
		return -1, "", err
	}

	return fd, "/proc/self/fd/" + strconv.Itoa(fd), nil
}

func ExecMemfd(name string, data []byte, argv []string) error {
	fd, path, err := CreateMemfdExecutable(name, data)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	if len(argv) == 0 {
		argv = []string{name}
	}
	return unix.Exec(path, argv, os.Environ())
}

func writeAll(fd int, data []byte) error {
	for len(data) > 0 {
		n, err := unix.Write(fd, data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
