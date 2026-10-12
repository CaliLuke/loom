package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
)

type disclosureRemedyWrapper struct {
	err    error
	remedy *loom.ErrorRemedy
}

func (e disclosureRemedyWrapper) Error() string {
	return "private wrapper: " + e.err.Error()
}

func (e disclosureRemedyWrapper) Unwrap() error {
	return e.err
}

func (e disclosureRemedyWrapper) LoomErrorRemedy() *loom.ErrorRemedy {
	return e.remedy
}

func TestServerErrorDisclosure(t *testing.T) {
	secret := errors.New("private database password")
	public := loom.NewServiceError(secret, "denied", false, false, true)
	public.Message = "public detail"
	safe := loom.WithErrorRemedy(loom.NewServiceError(secret, "denied", false, false, true), &loom.ErrorRemedy{SafeMessage: "safe detail"})
	for _, tc := range []struct {
		name   string
		err    error
		want   string
		code   codes.Code
		native bool
	}{
		{"plain", secret, "internal server error", codes.Unknown, false},
		{"wrapped plain", fmt.Errorf("private wrapper: %w", secret), "internal server error", codes.Unknown, false},
		{"public", public, "public detail", codes.Internal, false},
		{"wrapped public", fmt.Errorf("private wrapper: %w", public), "public detail", codes.Internal, false},
		{"safe", safe, "safe detail", codes.Internal, false},
		{"join", errors.Join(public, secret), "internal server error", codes.Unknown, false},
		{"reversed join", errors.Join(secret, public), "internal server error", codes.Unknown, false},
		{"safe branch", errors.Join(safe, secret), "internal server error", codes.Unknown, false},
		{"cancel", fmt.Errorf("private wrapper: %w", context.Canceled), "context canceled", codes.Canceled, true},
		{"deadline", fmt.Errorf("private wrapper: %w", context.DeadlineExceeded), "context deadline exceeded", codes.DeadlineExceeded, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.err.Error()
			for _, encoded := range []error{EncodeError(tc.err), EncodeServerError(tc.err, nil)} {
				st := status.Convert(encoded)
				require.Equal(t, tc.code, st.Code())
				require.Equal(t, tc.want, st.Message())
				if tc.native {
					require.Empty(t, st.Details())
				} else {
					detail, ok := DecodeError(encoded).(*loompb.ErrorResponse)
					require.True(t, ok)
					require.Equal(t, tc.want, detail.Msg)
				}
			}
			require.Equal(t, before, tc.err.Error())
		})
	}
	require.ErrorIs(t, public, secret)
}

func TestHistoryErrorDisclosure(t *testing.T) {
	first := loom.InvalidFormatError("password", "private password", loom.FormatEmail, errors.New("private parser detail"))
	second := loom.MissingFieldError("name", "body")
	merged := loom.MergeErrors(first, second)
	before := merged.Error()
	got := NewErrorResponse(merged)
	require.NotContains(t, got.String(), "private")
	require.Len(t, got.History, 2)
	require.Equal(t, loom.ErrorSafeMessage(first), got.History[0].Msg)
	require.Equal(t, loom.ErrorSafeMessage(second), got.History[1].Msg)
	require.Equal(t, before, merged.Error())

	unsafe := loom.MergeErrors(loom.Fault("private first"), loom.Fault("private second"))
	wrapped := disclosureRemedyWrapper{err: unsafe, remedy: &loom.ErrorRemedy{SafeMessage: "safe aggregate"}}
	before = unsafe.Error()
	got = NewErrorResponse(wrapped)
	require.Equal(t, "safe aggregate", got.Msg)
	require.Len(t, got.History, 2)
	for _, entry := range got.History {
		require.Equal(t, "safe aggregate", entry.Msg)
	}
	require.Equal(t, before, unsafe.Error())
}

func TestNestedHistoryErrorDisclosure(t *testing.T) {
	for _, tc := range []struct {
		name      string
		outer     *loom.ErrorRemedy
		entrySafe bool
	}{
		{"nil remedy", nil, false},
		{"code-only remedy", &loom.ErrorRemedy{Code: "retry"}, false},
		{"empty remedy", &loom.ErrorRemedy{}, false},
		{"nil remedy with safe entry", nil, true},
		{"code-only remedy with safe entry", &loom.ErrorRemedy{Code: "retry"}, true},
		{"empty remedy with safe entry", &loom.ErrorRemedy{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := loom.Fault("private first")
			second := loom.Fault("private second")
			secondMessage := "safe aggregate"
			if tc.entrySafe {
				second.Remedy = &loom.ErrorRemedy{SafeMessage: "safe entry"}
				secondMessage = "safe entry"
			}
			merged := loom.MergeErrors(first, second)
			safe := disclosureRemedyWrapper{err: merged, remedy: &loom.ErrorRemedy{SafeMessage: "safe aggregate"}}
			outer := disclosureRemedyWrapper{err: safe, remedy: tc.outer}
			before := outer.Error()
			for _, got := range []*loompb.ErrorResponse{
				NewErrorResponse(outer),
				DecodeError(EncodeServerError(outer, nil)).(*loompb.ErrorResponse),
			} {
				require.Equal(t, "safe aggregate", got.Msg)
				require.Len(t, got.History, 2)
				require.Equal(t, "safe aggregate", got.History[0].Msg)
				require.Equal(t, secondMessage, got.History[1].Msg)
			}
			require.Equal(t, before, outer.Error())
			require.Equal(t, "private first", first.Message)
			require.Equal(t, "private second", second.Message)
			require.ErrorIs(t, outer, merged)
		})
	}
}

func TestExplicitStatusErrorDisclosure(t *testing.T) {
	explicit := status.Error(codes.PermissionDenied, "explicit public protocol text")
	require.Equal(t, "explicit public protocol text", status.Convert(EncodeServerError(explicit, nil)).Message())
	require.Equal(t, "explicit constructor text", status.Convert(NewStatusError(codes.Unknown, errors.New("explicit constructor text"))).Message())
}
