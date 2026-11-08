//go:build !unix

package openapiimport

import "os"

// Non-Unix platforms use the regular-file preflight and descriptor validation.
const descriptionOpenFlags = os.O_RDONLY
