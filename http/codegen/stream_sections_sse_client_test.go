package codegen

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderSSEParseAssignment(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "string")
		require.Contains(t, code, "\tevent = dataContent\n")
	})

	t.Run("bytes", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "[]byte")
		require.Contains(t, code, "\tevent = []byte(dataContent)\n")
	})

	t.Run("int", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "int")
		require.Contains(t, code, "strconv.Atoi(dataContent)")
		require.Contains(t, code, "\tevent = v\n")
	})

	t.Run("default decode", func(t *testing.T) {
		code := renderSSEParseAssignment("event", "*Example")
		require.Contains(t, code, "strings.NewReader(dataContent)")
		require.Contains(t, code, "s.decoder(respBody).Decode(&event)")
	})
}

func TestSSEClientEmitterSectionStillRendersCoreHelpers(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) readEvent(ctx context.Context) ([]byte, error) {")
	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) checkBuffer() ([]byte, bool) {")
	require.Contains(t, code, "func (s *SSEObjectMethodStreamImpl) processEvent(eventData []byte)")
	require.True(t, strings.Index(code, "func (s *SSEObjectMethodStreamImpl) readEvent") < strings.Index(code, "func (s *SSEObjectMethodStreamImpl) processEvent"))
}

func TestSSEClientEmitterDoesNotHoldLockAcrossBlockingRead(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "readLock sync.Mutex")
	require.Contains(t, code, "s.readLock.Lock()\n\tdefer s.readLock.Unlock()")
	require.Contains(t, code, "case <-ctx.Done():\n\t\t\treturn nil, ctx.Err()")
	require.NotContains(t, code, "case <-ctx.Done():\n\t\t\tif len(eventData) > 0 {")
	require.Contains(t, code, "body := s.resp.Body")
	require.Contains(t, code, "n, err := body.Read(buf)")
	require.Contains(t, code, "case <-ctx.Done():\n\t\t\tselect {\n\t\t\tcase result := <-readc:")
	require.Contains(t, code, "default:\n\t\t\t\t_ = s.Close()\n\t\t\t\treturn nil, ctx.Err()")
	require.NotContains(t, code, "if ctx.Err() != nil {\n\t\t\treturn nil, ctx.Err()\n\t\t}")
	require.NotContains(t, code, "n, err := s.resp.Body.Read(buf)")

	read := strings.Index(code, "n, err := body.Read(buf)")
	unlock := strings.LastIndex(code[:read], "s.lock.Unlock()")
	lock := strings.LastIndex(code[:read], "s.lock.Lock()")
	require.Greater(t, unlock, lock, "readEvent must release the mutex before blocking on Body.Read")

	closeBody := strings.Index(code, "return body.Close()")
	require.NotEqual(t, -1, closeBody)
	closeUnlock := strings.LastIndex(code[:closeBody], "s.lock.Unlock()")
	closeLock := strings.LastIndex(code[:closeBody], "s.lock.Lock()")
	require.Greater(t, closeUnlock, closeLock, "Close must release the mutex before closing the body")
}

func TestSSEClientEmitterReturnsEOFFromRecv(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		testdata.SSEObjectDSL()
	})
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)
	sseFile := findFileWithSection(t, files, "client-sse")
	code := renderedSectionSource(t, sseFile.Section("client-sse")[0])

	require.Contains(t, code, "if errors.Is(err, io.EOF) {\n\t\t\ts.Close()\n\t\t\treturn event, io.EOF\n\t\t}")
	require.NotContains(t, code, "err = nil")
}
