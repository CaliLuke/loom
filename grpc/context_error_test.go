package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestContextErrorContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		code   codes.Code
		native bool
	}{
		{"cancel", context.Canceled, codes.Canceled, true},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded, true},
		{"wrapped", fmt.Errorf("operation: %w", context.Canceled), codes.Canceled, true},
		{"single join", errors.Join(context.DeadlineExceeded), codes.DeadlineExceeded, true},
		{"mixed", errors.Join(context.Canceled, errors.New("cleanup")), codes.Unknown, false},
		{"reversed", errors.Join(errors.New("cleanup"), context.Canceled), codes.Unknown, false},
		{"unanimous", errors.Join(context.Canceled, context.Canceled), codes.Canceled, false},
		{"owner", loom.NewServiceError(context.Canceled, "fault", false, false, true), codes.Internal, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, got := range []error{EncodeError(tc.err), EncodeServerError(tc.err, nil)} {
				require.Equal(t, tc.code, status.Code(got))
				require.Equal(t, tc.err.Error(), status.Convert(got).Message())
				if tc.native {
					require.Empty(t, status.Convert(got).Details())
				}
			}
		})
	}
}
