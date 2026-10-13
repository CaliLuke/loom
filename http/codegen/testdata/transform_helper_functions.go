package testdata

var PayloadBodyUserInnerDefaultTransformCode1 = `// unmarshalInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType builds a
// value of type *servicebodyuserinnerdefault.InnerType from a value of type
// *InnerTypeRequestBody.
func unmarshalInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType(v *InnerTypeRequestBody) *servicebodyuserinnerdefault.InnerType {
	if v == nil {
		return nil
	}
	res := &servicebodyuserinnerdefault.InnerType{}
	if actual, ok := v.A.Value(); ok {
		resAValue := actual

		res.A = resAValue
	}
	if actual, ok := v.B.Value(); ok {
		resBValue := actual

		res.B = resBValue
	} else {
		res.B = "patternb"
	}

	return res
}
`

var PayloadBodyInlineRecursiveUserTransformCode1 = `// unmarshalPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType
// builds a value of type *servicebodyinlinerecursiveuser.PayloadType from a
// value of type *PayloadTypeRequestBody.
func unmarshalPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType(v *PayloadTypeRequestBody) *servicebodyinlinerecursiveuser.PayloadType {
	if v == nil {
		return nil
	}
	res := &servicebodyinlinerecursiveuser.PayloadType{
		A: *v.A,
	}
	if actual, ok := v.B.Value(); ok {
		resBValue := actual

		res.B = &resBValue
	}
	res.C = unmarshalPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType(v.C)

	return res
}
`

var PayloadBodyUserInnerDefaultTransformCodeCLI1 = `// unmarshalCLIInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType
// builds a value of type *servicebodyuserinnerdefault.InnerType from a value
// of type *CLIInnerTypeRequestBody.
func unmarshalCLIInnerTypeRequestBodyToServicebodyuserinnerdefaultInnerType(v *CLIInnerTypeRequestBody) *servicebodyuserinnerdefault.InnerType {
	if v == nil {
		return nil
	}
	res := &servicebodyuserinnerdefault.InnerType{}
	if actual, ok := v.A.Value(); ok {
		resAValue := actual

		res.A = resAValue
	}
	if actual, ok := v.B.Value(); ok {
		resBValue := actual

		res.B = resBValue
	} else {
		res.B = "patternb"
	}

	return res
}
`

var PayloadBodyUserInnerDefaultTransformCodeCLI2 = `// marshalServicebodyuserinnerdefaultInnerTypeToInnerTypeRequestBody builds a
// value of type *InnerTypeRequestBody from a value of type
// *servicebodyuserinnerdefault.InnerType.
func marshalServicebodyuserinnerdefaultInnerTypeToInnerTypeRequestBody(v *servicebodyuserinnerdefault.InnerType) *InnerTypeRequestBody {
	if v == nil {
		return nil
	}
	res := &InnerTypeRequestBody{
		A: v.A,
		B: v.B,
	}

	return res
}
`

var PayloadBodyInlineRecursiveUserTransformCodeCLI1 = `// unmarshalCLIPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType
// builds a value of type *servicebodyinlinerecursiveuser.PayloadType from a
// value of type *CLIPayloadTypeRequestBody.
func unmarshalCLIPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType(v *CLIPayloadTypeRequestBody) *servicebodyinlinerecursiveuser.PayloadType {
	if v == nil {
		return nil
	}
	res := &servicebodyinlinerecursiveuser.PayloadType{
		A: *v.A,
	}
	if actual, ok := v.B.Value(); ok {
		resBValue := actual

		res.B = &resBValue
	}
	res.C = unmarshalCLIPayloadTypeRequestBodyToServicebodyinlinerecursiveuserPayloadType(v.C)

	return res
}
`

var PayloadBodyInlineRecursiveUserTransformCodeCLI2 = `// marshalServicebodyinlinerecursiveuserPayloadTypeToPayloadTypeRequestBody
// builds a value of type *PayloadTypeRequestBody from a value of type
// *servicebodyinlinerecursiveuser.PayloadType.
func marshalServicebodyinlinerecursiveuserPayloadTypeToPayloadTypeRequestBody(v *servicebodyinlinerecursiveuser.PayloadType) *PayloadTypeRequestBody {
	res := &PayloadTypeRequestBody{
		A: v.A,
		B: v.B,
	}
	if v.C != nil {
		res.C = marshalServicebodyinlinerecursiveuserPayloadTypeToPayloadTypeRequestBody(v.C)
	}

	return res
}
`
