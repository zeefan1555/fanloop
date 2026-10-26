//go:build !darwin && !linux

package currentlayout

import "fmt"

func exchange(left, right string) error {
	return fmt.Errorf("atomic current migration is unsupported on this platform")
}
