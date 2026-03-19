{{ printf "%sClientStream implements the %s.%sClientStream interface using Server-Sent Events." .Method.VarName .ServicePkgName .Method.VarName | comment }}
type {{ .Method.VarName }}ClientStream struct {
	resp    *http.Response  // HTTP response object
	reader  *bufio.Reader   // Buffered reader for SSE parsing
	decoder func(*http.Response) goahttp.Decoder  // User-provided decoder
	closed  bool            // Whether the stream has been closed
	lock    sync.Mutex      // Mutex to protect state
}

// readSSEEvent reads a single SSE event from the stream.
func (s *{{ .Method.VarName }}ClientStream) readSSEEvent() ([]byte, error) {
	var event bytes.Buffer

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && event.Len() > 0 {
				return event.Bytes(), nil
			}
			return nil, err
		}

		event.WriteString(line)

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if event.Len() > 0 {
				return event.Bytes(), nil
			}
			continue
		}
	}
}

{{ comment .Method.ClientStream.RecvDesc }}
func (s *{{ .Method.VarName }}ClientStream) {{ .Method.ClientStream.RecvName }}(ctx context.Context) ({{ .Result.Ref }}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	
	var zero {{ .Result.Ref }}
	
	if s.closed {
		return zero, io.EOF
	}
	
	for {
		rawEvent, err := s.readSSEEvent()
		if err != nil {
			s.closed = true
			return zero, err
		}

		parsedEvent, err := goahttp.ParseSSEEvent(rawEvent)
		if err != nil {
			s.closed = true
			return zero, err
		}

		eventType, data := parsedEvent.Type, []byte(parsedEvent.Data)
		
		switch eventType {
		case "notification":
			// Parse JSON-RPC notification
			var notification struct {
				JSONRPC string          `json:"jsonrpc"`
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(data, &notification); err != nil {
				return zero, fmt.Errorf("failed to parse notification: %w", err)
			}
			
			// Validate notification
			if notification.JSONRPC != "2.0" {
				return zero, fmt.Errorf("invalid JSON-RPC version: %s", notification.JSONRPC)
			}
			
			if notification.Method != {{ printf "%q" .Method.Name }} {
				// Skip notifications for other methods
				continue
			}
			
			// Decode the result from params
			{{- if .Method.Result }}
			result, err := s.decodeResult(notification.Params)
			if err != nil {
				return zero, fmt.Errorf("failed to decode result: %w", err)
			}
			return result, nil
			{{- else }}
			// Method has no result
			return zero, nil
			{{- end }}
			
		case "response":
			// Final response - parse and return
			var response jsonrpc.Response
			if err := json.Unmarshal(data, &response); err != nil {
				return zero, fmt.Errorf("failed to parse response: %w", err)
			}
			
			if response.Error != nil {
				return zero, fmt.Errorf("JSON-RPC error %d: %s", response.Error.Code, response.Error.Message)
			}
			
			{{- if .Method.Result }}
			// Decode the final result
			if response.Result == nil {
				return zero, fmt.Errorf("missing result in response")
			}
			// Convert response.Result to json.RawMessage
			resultBytes, err := json.Marshal(response.Result)
			if err != nil {
				return zero, fmt.Errorf("failed to marshal result: %w", err)
			}
			result, err := s.decodeResult(json.RawMessage(resultBytes))
			if err != nil {
				return zero, fmt.Errorf("failed to decode final result: %w", err)
			}
			
			// Mark stream as closed after final response
			s.closed = true
			return result, nil
			{{- else }}
			// Method has no result
			s.closed = true
			return zero, nil
			{{- end }}
			
		case "error":
			// Error response
			var response jsonrpc.Response
			if err := json.Unmarshal(data, &response); err != nil {
				return zero, fmt.Errorf("failed to parse error response: %w", err)
			}
			
			s.closed = true
			if response.Error != nil {
				return zero, fmt.Errorf("JSON-RPC error %d: %s", response.Error.Code, response.Error.Message)
			}
			return zero, fmt.Errorf("unexpected error response")

		case "", "message":
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				return zero, fmt.Errorf("failed to parse message event: %w", err)
			}

			if _, ok := envelope["method"]; ok {
				// Parse generic JSON-RPC notification carried on the default/message SSE event.
				var notification struct {
					JSONRPC string          `json:"jsonrpc"`
					Method  string          `json:"method"`
					Params  json.RawMessage `json:"params"`
				}
				if err := json.Unmarshal(data, &notification); err != nil {
					return zero, fmt.Errorf("failed to parse notification: %w", err)
				}
				if notification.JSONRPC != "2.0" {
					return zero, fmt.Errorf("invalid JSON-RPC version: %s", notification.JSONRPC)
				}
				if notification.Method != {{ printf "%q" .Method.Name }} {
					continue
				}

				{{- if .Method.Result }}
				result, err := s.decodeResult(notification.Params)
				if err != nil {
					return zero, fmt.Errorf("failed to decode result: %w", err)
				}
				return result, nil
				{{- else }}
				return zero, nil
				{{- end }}
			}

			// Parse generic JSON-RPC response or error carried on the default/message SSE event.
			var response jsonrpc.Response
			if err := json.Unmarshal(data, &response); err != nil {
				return zero, fmt.Errorf("failed to parse response: %w", err)
			}
			if response.Error != nil {
				s.closed = true
				return zero, fmt.Errorf("JSON-RPC error %d: %s", response.Error.Code, response.Error.Message)
			}

			{{- if .Method.Result }}
			if response.Result == nil {
				return zero, fmt.Errorf("missing result in response")
			}
			resultBytes, err := json.Marshal(response.Result)
			if err != nil {
				return zero, fmt.Errorf("failed to marshal result: %w", err)
			}
			result, err := s.decodeResult(json.RawMessage(resultBytes))
			if err != nil {
				return zero, fmt.Errorf("failed to decode final result: %w", err)
			}
			s.closed = true
			return result, nil
			{{- else }}
			s.closed = true
			return zero, nil
			{{- end }}
			
		default:
			// Ignore unknown event types
			continue
		}
	}
}

{{- if .Method.Result }}
// decodeResult decodes JSON-RPC result data using the user-provided decoder
func (s *{{ .Method.VarName }}ClientStream) decodeResult(data json.RawMessage) ({{ .Result.Ref }}, error) {
	// Create minimal HTTP response with raw JSON data for user's decoder
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
	
	// Use the user-provided decoder to decode the result
	decoder := s.decoder(resp)
	var result {{ .Result.Ref }}
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	
	return result, nil
}
{{- end }}

{{ comment "Close closes the stream." }}
func (s *{{ .Method.VarName }}ClientStream) Close() error {
    s.lock.Lock()
    defer s.lock.Unlock()
    
    if !s.closed {
        s.closed = true
        if s.resp != nil && s.resp.Body != nil {
            return s.resp.Body.Close()
        }
    }
    return nil
}
