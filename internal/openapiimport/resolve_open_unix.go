//go:build unix

package openapiimport

import (
	"os"
	"syscall"
)

// Nonblocking open closes the stat/open race for FIFOs. The opened descriptor
// is checked for regular-file identity before any read.
const descriptionOpenFlags = os.O_RDONLY | syscall.O_NONBLOCK
