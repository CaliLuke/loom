package http

import (
	"fmt"
	"net/http"
)

// AddRequestCookie validates name and value using the standard HTTP cookie
// grammar, then adds them to req. It preserves the supplied value without
// escaping or sanitizing it and leaves request headers unchanged on rejection.
// Response-only attributes and deployment policies do not apply to requests.
func AddRequestCookie(req *http.Request, name, value string) error {
	cookie := &http.Cookie{Name: name, Value: value}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("validate request cookie %q: %w", name, err)
	}
	req.AddCookie(cookie)
	return nil
}
