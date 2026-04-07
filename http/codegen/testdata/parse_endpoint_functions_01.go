package testdata


var MultiNoPayloadParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		serviceMultiNoPayload1Flags = flag.NewFlagSet("service-multi-no-payload1", flag.ContinueOnError)

		serviceMultiNoPayload1MethodServiceNoPayload11Flags = flag.NewFlagSet("method-service-no-payload11", flag.ExitOnError)

		serviceMultiNoPayload1MethodServiceNoPayload12Flags = flag.NewFlagSet("method-service-no-payload12", flag.ExitOnError)

		serviceMultiNoPayload2Flags = flag.NewFlagSet("service-multi-no-payload2", flag.ContinueOnError)

		serviceMultiNoPayload2MethodServiceNoPayload21Flags = flag.NewFlagSet("method-service-no-payload21", flag.ExitOnError)

		serviceMultiNoPayload2MethodServiceNoPayload22Flags = flag.NewFlagSet("method-service-no-payload22", flag.ExitOnError)
	)
	serviceMultiNoPayload1Flags.Usage = serviceMultiNoPayload1Usage
	serviceMultiNoPayload1MethodServiceNoPayload11Flags.Usage = serviceMultiNoPayload1MethodServiceNoPayload11Usage
	serviceMultiNoPayload1MethodServiceNoPayload12Flags.Usage = serviceMultiNoPayload1MethodServiceNoPayload12Usage

	serviceMultiNoPayload2Flags.Usage = serviceMultiNoPayload2Usage
	serviceMultiNoPayload2MethodServiceNoPayload21Flags.Usage = serviceMultiNoPayload2MethodServiceNoPayload21Usage
	serviceMultiNoPayload2MethodServiceNoPayload22Flags.Usage = serviceMultiNoPayload2MethodServiceNoPayload22Usage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "service-multi-no-payload1":
			svcf = serviceMultiNoPayload1Flags
		case "service-multi-no-payload2":
			svcf = serviceMultiNoPayload2Flags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "service-multi-no-payload1":
			switch epn {
			case "method-service-no-payload11":
				epf = serviceMultiNoPayload1MethodServiceNoPayload11Flags

			case "method-service-no-payload12":
				epf = serviceMultiNoPayload1MethodServiceNoPayload12Flags

			}

		case "service-multi-no-payload2":
			switch epn {
			case "method-service-no-payload21":
				epf = serviceMultiNoPayload2MethodServiceNoPayload21Flags

			case "method-service-no-payload22":
				epf = serviceMultiNoPayload2MethodServiceNoPayload22Flags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "service-multi-no-payload1":
			c := servicemultinopayload1c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-service-no-payload11":
				endpoint = c.MethodServiceNoPayload11()
			case "method-service-no-payload12":
				endpoint = c.MethodServiceNoPayload12()
			}
		case "service-multi-no-payload2":
			c := servicemultinopayload2c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-service-no-payload21":
				endpoint = c.MethodServiceNoPayload21()
			case "method-service-no-payload22":
				endpoint = c.MethodServiceNoPayload22()
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var MultiSimpleParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		serviceMultiSimple1Flags = flag.NewFlagSet("service-multi-simple1", flag.ContinueOnError)

		serviceMultiSimple1MethodMultiSimpleNoPayloadFlags = flag.NewFlagSet("method-multi-simple-no-payload", flag.ExitOnError)

		serviceMultiSimple1MethodMultiSimplePayloadFlags    = flag.NewFlagSet("method-multi-simple-payload", flag.ExitOnError)
		serviceMultiSimple1MethodMultiSimplePayloadBodyFlag = serviceMultiSimple1MethodMultiSimplePayloadFlags.String("body", "REQUIRED", "")

		serviceMultiSimple2Flags = flag.NewFlagSet("service-multi-simple2", flag.ContinueOnError)

		serviceMultiSimple2MethodMultiSimpleNoPayloadFlags = flag.NewFlagSet("method-multi-simple-no-payload", flag.ExitOnError)

		serviceMultiSimple2MethodMultiSimplePayloadFlags    = flag.NewFlagSet("method-multi-simple-payload", flag.ExitOnError)
		serviceMultiSimple2MethodMultiSimplePayloadBodyFlag = serviceMultiSimple2MethodMultiSimplePayloadFlags.String("body", "REQUIRED", "")
	)
	serviceMultiSimple1Flags.Usage = serviceMultiSimple1Usage
	serviceMultiSimple1MethodMultiSimpleNoPayloadFlags.Usage = serviceMultiSimple1MethodMultiSimpleNoPayloadUsage
	serviceMultiSimple1MethodMultiSimplePayloadFlags.Usage = serviceMultiSimple1MethodMultiSimplePayloadUsage

	serviceMultiSimple2Flags.Usage = serviceMultiSimple2Usage
	serviceMultiSimple2MethodMultiSimpleNoPayloadFlags.Usage = serviceMultiSimple2MethodMultiSimpleNoPayloadUsage
	serviceMultiSimple2MethodMultiSimplePayloadFlags.Usage = serviceMultiSimple2MethodMultiSimplePayloadUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "service-multi-simple1":
			svcf = serviceMultiSimple1Flags
		case "service-multi-simple2":
			svcf = serviceMultiSimple2Flags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "service-multi-simple1":
			switch epn {
			case "method-multi-simple-no-payload":
				epf = serviceMultiSimple1MethodMultiSimpleNoPayloadFlags

			case "method-multi-simple-payload":
				epf = serviceMultiSimple1MethodMultiSimplePayloadFlags

			}

		case "service-multi-simple2":
			switch epn {
			case "method-multi-simple-no-payload":
				epf = serviceMultiSimple2MethodMultiSimpleNoPayloadFlags

			case "method-multi-simple-payload":
				epf = serviceMultiSimple2MethodMultiSimplePayloadFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "service-multi-simple1":
			c := servicemultisimple1c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-multi-simple-no-payload":
				endpoint = c.MethodMultiSimpleNoPayload()
			case "method-multi-simple-payload":
				endpoint = c.MethodMultiSimplePayload()
				data, err = servicemultisimple1c.BuildMethodMultiSimplePayloadPayload(*serviceMultiSimple1MethodMultiSimplePayloadBodyFlag)
			}
		case "service-multi-simple2":
			c := servicemultisimple2c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-multi-simple-no-payload":
				endpoint = c.MethodMultiSimpleNoPayload()
			case "method-multi-simple-payload":
				endpoint = c.MethodMultiSimplePayload()
				data, err = servicemultisimple2c.BuildMethodMultiSimplePayloadPayload(*serviceMultiSimple2MethodMultiSimplePayloadBodyFlag)
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var MultiRequiredPayloadParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		serviceMultiRequired1Flags = flag.NewFlagSet("service-multi-required1", flag.ContinueOnError)

		serviceMultiRequired1MethodMultiRequiredPayloadFlags    = flag.NewFlagSet("method-multi-required-payload", flag.ExitOnError)
		serviceMultiRequired1MethodMultiRequiredPayloadBodyFlag = serviceMultiRequired1MethodMultiRequiredPayloadFlags.String("body", "REQUIRED", "")

		serviceMultiRequired2Flags = flag.NewFlagSet("service-multi-required2", flag.ContinueOnError)

		serviceMultiRequired2MethodMultiRequiredNoPayloadFlags = flag.NewFlagSet("method-multi-required-no-payload", flag.ExitOnError)

		serviceMultiRequired2MethodMultiRequiredPayloadFlags = flag.NewFlagSet("method-multi-required-payload", flag.ExitOnError)
		serviceMultiRequired2MethodMultiRequiredPayloadAFlag = serviceMultiRequired2MethodMultiRequiredPayloadFlags.String("a", "REQUIRED", "")
	)
	serviceMultiRequired1Flags.Usage = serviceMultiRequired1Usage
	serviceMultiRequired1MethodMultiRequiredPayloadFlags.Usage = serviceMultiRequired1MethodMultiRequiredPayloadUsage

	serviceMultiRequired2Flags.Usage = serviceMultiRequired2Usage
	serviceMultiRequired2MethodMultiRequiredNoPayloadFlags.Usage = serviceMultiRequired2MethodMultiRequiredNoPayloadUsage
	serviceMultiRequired2MethodMultiRequiredPayloadFlags.Usage = serviceMultiRequired2MethodMultiRequiredPayloadUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "service-multi-required1":
			svcf = serviceMultiRequired1Flags
		case "service-multi-required2":
			svcf = serviceMultiRequired2Flags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "service-multi-required1":
			switch epn {
			case "method-multi-required-payload":
				epf = serviceMultiRequired1MethodMultiRequiredPayloadFlags

			}

		case "service-multi-required2":
			switch epn {
			case "method-multi-required-no-payload":
				epf = serviceMultiRequired2MethodMultiRequiredNoPayloadFlags

			case "method-multi-required-payload":
				epf = serviceMultiRequired2MethodMultiRequiredPayloadFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "service-multi-required1":
			c := servicemultirequired1c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-multi-required-payload":
				endpoint = c.MethodMultiRequiredPayload()
				data, err = servicemultirequired1c.BuildMethodMultiRequiredPayloadPayload(*serviceMultiRequired1MethodMultiRequiredPayloadBodyFlag)
			}
		case "service-multi-required2":
			c := servicemultirequired2c.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-multi-required-no-payload":
				endpoint = c.MethodMultiRequiredNoPayload()
			case "method-multi-required-payload":
				endpoint = c.MethodMultiRequiredPayload()
				data, err = servicemultirequired2c.BuildMethodMultiRequiredPayloadPayload(*serviceMultiRequired2MethodMultiRequiredPayloadAFlag)
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var SkipRequestBodyEncodeDecodeParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		skipRequestBodyEncodeDecodeFlags = flag.NewFlagSet("skip-request-body-encode-decode", flag.ContinueOnError)

		skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodFlags      = flag.NewFlagSet("skip-request-body-encode-decode-method", flag.ExitOnError)
		skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodStreamFlag = skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodFlags.String("stream", "REQUIRED", "path to file containing the streamed request body")
	)
	skipRequestBodyEncodeDecodeFlags.Usage = skipRequestBodyEncodeDecodeUsage
	skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodFlags.Usage = skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "skip-request-body-encode-decode":
			svcf = skipRequestBodyEncodeDecodeFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "skip-request-body-encode-decode":
			switch epn {
			case "skip-request-body-encode-decode-method":
				epf = skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "skip-request-body-encode-decode":
			c := skiprequestbodyencodedecodec.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "skip-request-body-encode-decode-method":
				endpoint = c.SkipRequestBodyEncodeDecodeMethod()
				data, err = skiprequestbodyencodedecodec.BuildSkipRequestBodyEncodeDecodeMethodStreamPayload(*skipRequestBodyEncodeDecodeSkipRequestBodyEncodeDecodeMethodStreamFlag)
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var MultiParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		serviceMultiFlags = flag.NewFlagSet("service-multi", flag.ContinueOnError)

		serviceMultiMethodMultiNoPayloadFlags = flag.NewFlagSet("method-multi-no-payload", flag.ExitOnError)

		serviceMultiMethodMultiPayloadFlags    = flag.NewFlagSet("method-multi-payload", flag.ExitOnError)
		serviceMultiMethodMultiPayloadBodyFlag = serviceMultiMethodMultiPayloadFlags.String("body", "REQUIRED", "")
		serviceMultiMethodMultiPayloadBFlag    = serviceMultiMethodMultiPayloadFlags.String("b", "", "")
		serviceMultiMethodMultiPayloadAFlag    = serviceMultiMethodMultiPayloadFlags.String("a", "", "")
	)
	serviceMultiFlags.Usage = serviceMultiUsage
	serviceMultiMethodMultiNoPayloadFlags.Usage = serviceMultiMethodMultiNoPayloadUsage
	serviceMultiMethodMultiPayloadFlags.Usage = serviceMultiMethodMultiPayloadUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "service-multi":
			svcf = serviceMultiFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "service-multi":
			switch epn {
			case "method-multi-no-payload":
				epf = serviceMultiMethodMultiNoPayloadFlags

			case "method-multi-payload":
				epf = serviceMultiMethodMultiPayloadFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "service-multi":
			c := servicemultic.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-multi-no-payload":
				endpoint = c.MethodMultiNoPayload()
			case "method-multi-payload":
				endpoint = c.MethodMultiPayload()
				data, err = servicemultic.BuildMethodMultiPayloadPayload(*serviceMultiMethodMultiPayloadBodyFlag, *serviceMultiMethodMultiPayloadBFlag, *serviceMultiMethodMultiPayloadAFlag)
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var StreamingParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
	dialer loomhttp.Dialer,
	streamingServiceAConfigurer *streamingserviceac.ConnConfigurer,
	streamingServiceBConfigurer *streamingservicebc.ConnConfigurer,
) (loom.Endpoint, any, error) {
	var (
		streamingServiceAFlags = flag.NewFlagSet("streaming-service-a", flag.ContinueOnError)

		streamingServiceAMethodFlags = flag.NewFlagSet("method", flag.ExitOnError)

		streamingServiceBFlags = flag.NewFlagSet("streaming-service-b", flag.ContinueOnError)

		streamingServiceBMethodFlags = flag.NewFlagSet("method", flag.ExitOnError)
	)
	streamingServiceAFlags.Usage = streamingServiceAUsage
	streamingServiceAMethodFlags.Usage = streamingServiceAMethodUsage

	streamingServiceBFlags.Usage = streamingServiceBUsage
	streamingServiceBMethodFlags.Usage = streamingServiceBMethodUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "streaming-service-a":
			svcf = streamingServiceAFlags
		case "streaming-service-b":
			svcf = streamingServiceBFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "streaming-service-a":
			switch epn {
			case "method":
				epf = streamingServiceAMethodFlags

			}

		case "streaming-service-b":
			switch epn {
			case "method":
				epf = streamingServiceBMethodFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "streaming-service-a":
			c := streamingserviceac.NewClient(scheme, host, doer, enc, dec, restore, dialer, streamingServiceAConfigurer)
			switch epn {
			case "method":
				endpoint = c.Method()
			}
		case "streaming-service-b":
			c := streamingservicebc.NewClient(scheme, host, doer, enc, dec, restore, dialer, streamingServiceBConfigurer)
			switch epn {
			case "method":
				endpoint = c.Method()
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`

var MultiSimpleBuildCode = `// BuildMethodMultiSimplePayloadPayload builds the payload for the
// ServiceMultiSimple1 MethodMultiSimplePayload endpoint from CLI flags.
func BuildMethodMultiSimplePayloadPayload(serviceMultiSimple1MethodMultiSimplePayloadBody string) (*servicemultisimple1.MethodMultiSimplePayloadPayload, error) {
	var err error
	var body MethodMultiSimplePayloadRequestBody
	{
		err = json.Unmarshal([]byte(serviceMultiSimple1MethodMultiSimplePayloadBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"a\": false\n   }'")
		}
	}
	v := &servicemultisimple1.MethodMultiSimplePayloadPayload{
		A: body.A,
	}

	return v, nil
}
`

var MultiBuildCode = `// BuildMethodMultiPayloadPayload builds the payload for the ServiceMulti
// MethodMultiPayload endpoint from CLI flags.
func BuildMethodMultiPayloadPayload(serviceMultiMethodMultiPayloadBody string, serviceMultiMethodMultiPayloadB string, serviceMultiMethodMultiPayloadA string) (*servicemulti.MethodMultiPayloadPayload, error) {
	var err error
	var body MethodMultiPayloadRequestBody
	{
		err = json.Unmarshal([]byte(serviceMultiMethodMultiPayloadBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"c\": {\n         \"att\": false,\n         \"att10\": \"Aspernatur quo error explicabo pariatur.\",\n         \"att11\": \"Q3VtcXVlIHZvbHVwdGF0ZW0u\",\n         \"att12\": \"Distinctio aliquam nihil blanditiis ut.\",\n         \"att13\": [\n            \"Nihil excepturi deserunt quasi omnis sed.\",\n            \"Sit maiores aperiam autem non ea rem.\"\n         ],\n         \"att14\": {\n            \"Excepturi totam.\": \"Ut aut facilis vel ipsam.\",\n            \"Minima et aut non sunt consequuntur.\": \"Et consequuntur porro quasi.\",\n            \"Quis voluptates quaerat et temporibus facere.\": \"Ipsam eaque sunt maxime suscipit.\"\n         },\n         \"att15\": {\n            \"inline\": \"Ea alias repellat nobis veritatis.\"\n         },\n         \"att2\": 3504438334001971349,\n         \"att3\": 2005839040,\n         \"att4\": 5845720715558772393,\n         \"att5\": 12124006045301819638,\n         \"att6\": 3731236027,\n         \"att7\": 10708117302649141570,\n         \"att8\": 0.11815318,\n         \"att9\": 0.30907290919538355\n      }\n   }'")
		}
	}
	var b *string
	{
		if serviceMultiMethodMultiPayloadB != "" {
			b = &serviceMultiMethodMultiPayloadB
		}
	}
	var a *bool
	{
		if serviceMultiMethodMultiPayloadA != "" {
			var val bool
			val, err = strconv.ParseBool(serviceMultiMethodMultiPayloadA)
			a = &val
			if err != nil {
				return nil, fmt.Errorf("invalid value for a, must be BOOL")
			}
		}
	}
	v := &servicemulti.MethodMultiPayloadPayload{}
	if body.C != nil {
		v.C = marshalUserTypeRequestBodyToServicemultiUserType(body.C)
	}
	v.B = b
	v.A = a

	return v, nil
}
`

var QueryBoolBuildCode = `// BuildMethodQueryBoolPayload builds the payload for the ServiceQueryBool
// MethodQueryBool endpoint from CLI flags.
func BuildMethodQueryBoolPayload(serviceQueryBoolMethodQueryBoolQ string) (*servicequerybool.MethodQueryBoolPayload, error) {
	var err error
	var q *bool
	{
		if serviceQueryBoolMethodQueryBoolQ != "" {
			var val bool
			val, err = strconv.ParseBool(serviceQueryBoolMethodQueryBoolQ)
			q = &val
			if err != nil {
				return nil, fmt.Errorf("invalid value for q, must be BOOL")
			}
		}
	}
	v := &servicequerybool.MethodQueryBoolPayload{}
	v.Q = q

	return v, nil
}
`

var BodyQueryPathObjectBuildCode = `// BuildMethodBodyQueryPathObjectPayload builds the payload for the
// ServiceBodyQueryPathObject MethodBodyQueryPathObject endpoint from CLI flags.
func BuildMethodBodyQueryPathObjectPayload(serviceBodyQueryPathObjectMethodBodyQueryPathObjectBody string, serviceBodyQueryPathObjectMethodBodyQueryPathObjectC2 string, serviceBodyQueryPathObjectMethodBodyQueryPathObjectB string) (*servicebodyquerypathobject.MethodBodyQueryPathObjectPayload, error) {
	var err error
	var body MethodBodyQueryPathObjectRequestBody
	{
		err = json.Unmarshal([]byte(serviceBodyQueryPathObjectMethodBodyQueryPathObjectBody), &body)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON for body, \nerror: %s, \nexample of valid JSON:\n%s", err, "'{\n      \"a\": \"Ullam aut.\"\n   }'")
		}
	}
	var c2 string
	{
		c2 = serviceBodyQueryPathObjectMethodBodyQueryPathObjectC2
	}
	var b *string
	{
		if serviceBodyQueryPathObjectMethodBodyQueryPathObjectB != "" {
			b = &serviceBodyQueryPathObjectMethodBodyQueryPathObjectB
		}
	}
	v := &servicebodyquerypathobject.MethodBodyQueryPathObjectPayload{
		A: body.A,
	}
	v.C = &c2
	v.B = b

	return v, nil
}
`

var ParamValidateBuildCode = `// BuildMethodParamValidatePayload builds the payload for the
// ServiceParamValidate MethodParamValidate endpoint from CLI flags.
func BuildMethodParamValidatePayload(serviceParamValidateMethodParamValidateA string) (*serviceparamvalidate.MethodParamValidatePayload, error) {
	var err error
	var a *int
	{
		if serviceParamValidateMethodParamValidateA != "" {
			var v int64
			v, err = strconv.ParseInt(serviceParamValidateMethodParamValidateA, 10, strconv.IntSize)
			val := int(v)
			a = &val
			if err != nil {
				return nil, fmt.Errorf("invalid value for a, must be INT")
			}
			if *a < 1 {
				err = loom.MergeErrors(err, loom.InvalidRangeError("a", *a, 1, true))
			}
			if err != nil {
				return nil, err
			}
		}
	}
	v := &serviceparamvalidate.MethodParamValidatePayload{}
	v.A = a

	return v, nil
}
`

var PayloadPrimitiveTypeParseCode = `// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer loomhttp.Doer,
	enc func(*http.Request) loomhttp.Encoder,
	dec func(*http.Response) loomhttp.Decoder,
	restore bool,
) (loom.Endpoint, any, error) {
	var (
		serviceBodyPrimitiveBoolValidateFlags = flag.NewFlagSet("service-body-primitive-bool-validate", flag.ContinueOnError)

		serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidateFlags = flag.NewFlagSet("method-body-primitive-bool-validate", flag.ExitOnError)
		serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidatePFlag = serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidateFlags.String("p", "REQUIRED", "bool is the payload type of the ServiceBodyPrimitiveBoolValidate service MethodBodyPrimitiveBoolValidate method.")
	)
	serviceBodyPrimitiveBoolValidateFlags.Usage = serviceBodyPrimitiveBoolValidateUsage
	serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidateFlags.Usage = serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidateUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "service-body-primitive-bool-validate":
			svcf = serviceBodyPrimitiveBoolValidateFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "service-body-primitive-bool-validate":
			switch epn {
			case "method-body-primitive-bool-validate":
				epf = serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidateFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint loom.Endpoint
		err      error
	)
	{
		switch svcn {
		case "service-body-primitive-bool-validate":
			c := servicebodyprimitiveboolvalidatec.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "method-body-primitive-bool-validate":
				endpoint = c.MethodBodyPrimitiveBoolValidate()
				var err error
				data, err = strconv.ParseBool(*serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidatePFlag)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid value for serviceBodyPrimitiveBoolValidateMethodBodyPrimitiveBoolValidatePFlag, must be BOOL")
				}
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}
`


