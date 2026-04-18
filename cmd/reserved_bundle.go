package cmd

import (
	"fmt"

	"skl/internal/library"
)

func rejectReservedBundle(name string) error {
	if name != library.ReservedInboxBundle {
		return nil
	}
	return fmt.Errorf("bundle %q is reserved; assign skills to another bundle to move them out of %s", name, name)
}
