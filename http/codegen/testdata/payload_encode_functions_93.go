package testdata


var PayloadHeaderCustomNameEncodeCode = `// EncodeMethodHeaderCustomNameRequest returns an encoder for requests sent to
// the ServiceHeaderCustomName MethodHeaderCustomName server.
func EncodeMethodHeaderCustomNameRequest(encoder func(*http.Request) loomhttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*serviceheadercustomname.MethodHeaderCustomNamePayload)
		if !ok {
			return loomhttp.ErrInvalidType("ServiceHeaderCustomName", "MethodHeaderCustomName", "*serviceheadercustomname.MethodHeaderCustomNamePayload", v)
		}
		if p.Header != nil {
			head := *p.Header
			req.Header.Set("h", head)
		}
		return nil
	}
}
`


