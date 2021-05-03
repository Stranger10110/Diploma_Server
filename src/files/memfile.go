package files

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// Linux 3.17+ is needed.
// memfile takes a file name used, and the byte slice
// containing data the file should contain.
//
// name does not need to be unique, as it's used only
// for debugging purposes.
//
// It is up to the caller to close the returned descriptor.
func MemFile(name string, b []byte, fileMode string) (int, string, error) {
	fd, err := unix.MemfdCreate(name, FileMode[fileMode])
	if err != nil {
		return 0, "", fmt.Errorf("MemfdCreate: %v", err)
	}

	bLength := len(b)
	if bLength != 0 {
		err = unix.Ftruncate(fd, int64(bLength))
		if err != nil {
			return 0, "", fmt.Errorf("Ftruncate: %v", err)
		}

		data, err2 := unix.Mmap(fd, 0, bLength, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
		if err2 != nil {
			return 0, "", fmt.Errorf("Mmap: %v", err)
		}

		copy(data, b)

		err2 = unix.Munmap(data)
		if err2 != nil {
			return 0, "", fmt.Errorf("Munmap: %v", err)
		}
	}

	filePath := fmt.Sprintf("/proc/self/fd/%d", fd)
	return fd, filePath, nil
}
