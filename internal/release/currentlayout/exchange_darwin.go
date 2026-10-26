package currentlayout

import "golang.org/x/sys/unix"

func exchange(left, right string) error {
	return unix.RenameatxNp(unix.AT_FDCWD, left, unix.AT_FDCWD, right, unix.RENAME_SWAP)
}
