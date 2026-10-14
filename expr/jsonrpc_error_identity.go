package expr

// JSONRPCErrorSelector returns the effective ErrorName path in the designed
// error data. Nil means no retained discriminator; an empty nonnil path selects
// the whole body. The result is detached and uses actual JSON member names.
func (e *HTTPErrorExpr) JSONRPCErrorSelector() []string {
	if e == nil || e.ErrorExpr == nil || e.Response == nil || e.Response.Body == nil {
		return nil
	}
	source := AsObject(e.Type)
	if source == nil {
		return nil
	}
	for _, field := range *source {
		if _, marked := field.Attribute.Meta["struct:error:name"]; !marked {
			continue
		}
		body := e.Response.Body
		if origin, ok := body.Meta.Last("origin:attribute"); ok {
			if origin == AttributeName(field.Name) {
				return []string{}
			}
			return nil
		}
		if object := AsObject(body.Type); object != nil {
			for _, member := range *object {
				if AttributeName(member.Name) != AttributeName(field.Name) {
					continue
				}
				name := JSONFieldName(ElementName(member.Name), member.Attribute)
				if name != "-" {
					return []string{name}
				}
			}
		}
	}
	return nil
}
