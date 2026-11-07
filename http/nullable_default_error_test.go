package http

import (
	"context"
	"errors"
	"fmt"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type nativeResponseOwner struct {
	native *loom.ServiceError
}

func (e nativeResponseOwner) Error() string {
	return "public message"
}

func (e nativeResponseOwner) LoomErrorName() string {
	return "missing"
}

func (e nativeResponseOwner) LoomServiceError() *loom.ServiceError {
	return e.native
}

func TestProblemResponseNativeOwner(t *testing.T) {
	native := loom.NewServiceError(errors.New("private cause"), "missing", true, true, false)
	owner := nativeResponseOwner{native}
	for _, failure := range []error{owner, fmt.Errorf("wrapper: %w", owner)} {
		got := NewProblemResponse(context.Background(), failure, 409, "", "")
		require.Equal(t, "missing", got.Code)
		require.Equal(t, ProblemInstanceURI(native.ID), got.Instance)
		require.True(t, got.Timeout)
		require.True(t, got.Temporary)
		require.False(t, got.Fault)
		require.Equal(t, "public message", got.Detail)
	}
	for _, failure := range []error{nativeResponseOwner{}, errors.Join(owner, errors.New("other"))} {
		got := NewProblemResponse(context.Background(), failure, 0, "", "")
		require.Equal(t, "fault", got.Code)
		require.True(t, got.Fault)
	}
}
