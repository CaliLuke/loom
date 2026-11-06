package http

import (
	"fmt"
	"net/http"
)

// AddRequestCookie validates name and value using the verbatim request cookie
// grammar, then appends name=value to req without quoting or sanitizing it.
// Invalid values leave every request header unchanged. Response-only attributes
// and deployment policies do not apply to requests.
func AddRequestCookie(req *http.Request, name, value string) error {
	if req == nil {
		return fmt.Errorf("parameter %q: cookie request is nil", name)
	}
	if !cookieParamData(value) {
		return paramError(name, "value cannot be represented as cookie data")
	}
	if err := validateCookiePair(name, value); err != nil {
		return err
	}
	publishRequestCookies(req, []string{name + "=" + value})
	return nil
}
