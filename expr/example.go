package expr

import (
	"fmt"
	"math"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"
	"unicode"
)

const maxAttempts = 500 // Max number of retries to generate valid example.

var printablePatternTables = []*unicode.RangeTable{
	unicode.L,
	unicode.M,
	unicode.N,
	unicode.P,
	unicode.S,
}

// Example returns the example set on the attribute at design time. If there
// isn't such a value then Example computes a random value for the attribute
// using the given random value producer.
func (a *AttributeExpr) Example(r *ExampleGenerator) any {
	if ex := a.ExtractUserExamples(); len(ex) > 0 {
		// Return the last item in the slice so that examples can be overridden
		// in the DSL. Overridden examples are always appended to the UserExamples
		// slice.
		return ex[len(ex)-1].Value
	}

	if r.Randomizer == nil {
		return nil
	}

	value, ok := a.Meta.Last("openapi:generate")
	if ok && value == "false" {
		return nil
	}

	value, ok = a.Meta.Last("openapi:example")
	if ok && value == "false" {
		return nil
	}

	// enum should dominate, because the potential "examples" are fixed
	if hasEnumValidation(a) {
		return byEnum(a, r)
	}
	return a.generatedExample(r)
}

func (a *AttributeExpr) generatedExample(r *ExampleGenerator) any {
	// loop until a satisfying example is generated
	var (
		hasFormat  = hasFormatValidation(a)
		hasPattern = hasPatternValidation(a)
		hasMinMax  = hasMinMaxValidation(a)
		hasLength  = hasLengthValidation(a)
		attempts   = 0
	)
	if hasLength && !hasFormat && !hasPattern {
		return byLength(a, r)
	}
	for attempts < maxAttempts {
		attempts++
		example := a.generatedCandidate(r, hasFormat, hasPattern, hasMinMax, hasLength)
		if example == nil {
			return nil
		}
		if !checkLength(a, example) || !checkPattern(a, example) || !checkMinMaxValue(a, example) {
			continue
		}
		return example
	}
	return nil
}

func (a *AttributeExpr) generatedCandidate(
	r *ExampleGenerator,
	hasFormat, hasPattern, hasMinMax, hasLength bool,
) any {
	var example any
	if hasFormat {
		example = byFormat(a, r)
	}
	if hasPattern && example == nil {
		example = byPattern(a, r)
	}
	if hasMinMax && example == nil {
		example = byMinMax(a, r)
	}
	if example != nil {
		return example
	}
	if hasLength {
		return byLength(a, r)
	}
	return a.Type.Example(r)
}

func hasEnumValidation(a *AttributeExpr) bool {
	return a.Validation != nil && len(a.Validation.Values) > 0
}

func hasFormatValidation(a *AttributeExpr) bool {
	return a.Validation != nil && a.Validation.Format != ""
}

func hasPatternValidation(a *AttributeExpr) bool {
	return a.Validation != nil && a.Validation.Pattern != ""
}

func hasMinMaxValidation(a *AttributeExpr) bool {
	if a.Validation == nil {
		return false
	}
	return a.Validation.ExclusiveMinimum != nil ||
		a.Validation.Minimum != nil ||
		a.Validation.ExclusiveMaximum != nil ||
		a.Validation.Maximum != nil
}

// byFormat returns a random example based on the format the user asks.
func byFormat(a *AttributeExpr, r *ExampleGenerator) any {
	if !hasFormatValidation(a) {
		return nil
	}
	format := a.Validation.Format
	if res, ok := map[ValidationFormat]any{
		FormatEmail:    r.Email(),
		FormatHostname: r.Hostname(),
		FormatDate:     time.Unix(int64(r.Int())%1454957045, 0).UTC().Format(time.DateOnly), // to obtain a "fixed" rand
		FormatDateTime: time.Unix(int64(r.Int())%1454957045, 0).UTC().Format(time.RFC3339),  // to obtain a "fixed" rand
		FormatIPv4:     r.IPv4Address().String(),
		FormatIPv6:     r.IPv6Address().String(),
		FormatIP:       r.IPv4Address().String(),
		FormatURI:      r.URL(),
		FormatMAC: func() string {
			res, err := syntax.Parse(`([0-9A-F]{2}-){5}[0-9A-F]{2}`, 0)
			if err != nil {
				return "12-34-56-78-9A-BC"
			}
			return patgen(res, r)
		}(),
		FormatCIDR:    "192.168.100.14/24",
		FormatRegexp:  r.Characters(3) + ".*",
		FormatRFC1123: time.Unix(int64(r.Int())%1454957045, 0).UTC().Format(time.RFC1123), // to obtain a "fixed" rand
		FormatUUID:    r.UUID(),
		FormatJSON:    `{"name":"example","email":"mail@example.com"}`,
	}[format]; ok {
		return res
	}
	panic("Validation: unknown format '" + format + "'") // bug
}

// byPattern generates a random value that satisfies the pattern.
//
// Note: if multiple patterns are given, only one of them is used.
func byPattern(a *AttributeExpr, r *ExampleGenerator) any {
	if !hasPatternValidation(a) {
		return false
	}
	pattern := a.Validation.Pattern
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return r.Name()
	}
	return patgen(re.Simplify(), r)
}

func patgen(re *syntax.Regexp, r *ExampleGenerator) string {
	switch re.Op {
	case syntax.OpAlternate:
		i := r.Int() % len(re.Sub)
		return patgen(re.Sub[i], r)
	case syntax.OpCapture:
		return patgen(re.Sub[0], r)
	case syntax.OpConcat:
		var res strings.Builder
		for _, sub := range re.Sub {
			res.WriteString(patgen(sub, r))
		}
		return res.String()
	case syntax.OpLiteral:
		return string(re.Rune)
	case syntax.OpStar:
		var res strings.Builder
		count := r.Int() % 3
		for range count {
			res.WriteString(patgen(re.Sub[0], r))
		}
		return res.String()
	case syntax.OpPlus:
		var res strings.Builder
		count := r.Int()%2 + 1
		for range count {
			res.WriteString(patgen(re.Sub[0], r))
		}
		return res.String()
	case syntax.OpQuest:
		if r.Int()%2 == 0 {
			return patgen(re.Sub[0], r)
		}
		return ""
	case syntax.OpRepeat:
		var res strings.Builder
		for i := 0; i < re.Min; i++ {
			res.WriteString(patgen(re.Sub[0], r))
		}
		return res.String()
	case syntax.OpCharClass:
		return string(patternClassRune(re.Rune, r))
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return r.Characters(1)
	default:
		return ""
	}
}

func patternClassRune(ranges []rune, r *ExampleGenerator) rune {
	selection := r.Int()
	if candidate, ok := selectPatternClassRune(ranges, ' ', '~', selection); ok {
		return candidate
	}
	if candidate, ok := selectPrintablePatternClassRune(ranges, selection); ok {
		return candidate
	}
	candidate, ok := selectPatternClassRune(ranges, 0, unicode.MaxRune, selection)
	if !ok {
		return unicode.ReplacementChar
	}
	return candidate
}

func selectPrintablePatternClassRune(ranges []rune, selection int) (rune, bool) {
	count := 0
	for _, table := range printablePatternTables {
		for _, sequence := range table.R16 {
			count += countPatternClassSequence(
				ranges,
				rune(sequence.Lo),
				rune(sequence.Hi),
				rune(sequence.Stride),
			)
		}
		for _, sequence := range table.R32 {
			count += countPatternClassSequence(
				ranges,
				rune(sequence.Lo),
				rune(sequence.Hi),
				rune(sequence.Stride),
			)
		}
	}
	if count == 0 {
		return 0, false
	}

	offset := selection % count
	if offset < 0 {
		offset += count
	}
	for _, table := range printablePatternTables {
		for _, sequence := range table.R16 {
			if candidate, ok := selectPatternClassSequence(
				ranges,
				rune(sequence.Lo),
				rune(sequence.Hi),
				rune(sequence.Stride),
				&offset,
			); ok {
				return candidate, true
			}
		}
		for _, sequence := range table.R32 {
			if candidate, ok := selectPatternClassSequence(
				ranges,
				rune(sequence.Lo),
				rune(sequence.Hi),
				rune(sequence.Stride),
				&offset,
			); ok {
				return candidate, true
			}
		}
	}
	return 0, false
}

func countPatternClassSequence(ranges []rune, sequenceStart, sequenceEnd, stride rune) int {
	count := 0
	for i := 0; i+1 < len(ranges); i += 2 {
		first, last, ok := patternClassSequenceBounds(
			ranges[i],
			ranges[i+1],
			sequenceStart,
			sequenceEnd,
			stride,
		)
		if ok {
			count += int((last-first)/stride) + 1
		}
	}
	return count
}

func selectPatternClassSequence(
	ranges []rune,
	sequenceStart, sequenceEnd, stride rune,
	offset *int,
) (rune, bool) {
	for i := 0; i+1 < len(ranges); i += 2 {
		first, last, ok := patternClassSequenceBounds(
			ranges[i],
			ranges[i+1],
			sequenceStart,
			sequenceEnd,
			stride,
		)
		if !ok {
			continue
		}
		count := int((last-first)/stride) + 1
		if *offset < count {
			return first + rune(*offset)*stride, true
		}
		*offset -= count
	}
	return 0, false
}

func patternClassSequenceBounds(
	classStart, classEnd, sequenceStart, sequenceEnd, stride rune,
) (rune, rune, bool) {
	first := max(classStart, sequenceStart)
	last := min(classEnd, sequenceEnd)
	if first > last {
		return 0, 0, false
	}
	remainder := (first - sequenceStart) % stride
	if remainder != 0 {
		first += stride - remainder
	}
	return first, last, first <= last
}

func selectPatternClassRune(ranges []rune, minimum, maximum rune, selection int) (rune, bool) {
	count := 0
	for i := 0; i+1 < len(ranges); i += 2 {
		start := max(ranges[i], minimum)
		end := min(ranges[i+1], maximum)
		if start <= end {
			count += int(end-start) + 1
		}
	}
	if count == 0 {
		return 0, false
	}

	offset := selection % count
	if offset < 0 {
		offset += count
	}
	for i := 0; i+1 < len(ranges); i += 2 {
		start := max(ranges[i], minimum)
		end := min(ranges[i+1], maximum)
		if start > end {
			continue
		}
		width := int(end-start) + 1
		if offset < width {
			return start + rune(offset), true
		}
		offset -= width
	}
	return 0, false
}

func byMinMax(a *AttributeExpr, r *ExampleGenerator) any {
	if !hasMinMaxValidation(a) {
		return nil
	}
	minimum, maximum, sign := minMaxBounds(a)
	if math.IsInf(maximum, 1) {
		return randomMinOnlyValue(a.Type.Kind(), r, minimum, sign)
	}
	if minimum < maximum {
		return randomBoundedValue(a.Type.Kind(), r, minimum, maximum)
	}
	return minValueForKind(a.Type.Kind(), minimum)
}

func minMaxBounds(a *AttributeExpr) (float64, float64, int) {
	minimum := readMinimum(a)
	maximum := readMaximum(a)
	sign := 1
	if a.Validation.ExclusiveMinimum == nil && a.Validation.Minimum == nil {
		sign = -1
		minimum = maximum
		maximum = math.Inf(1)
	}
	return minimum, maximum, sign
}

func readMaximum(a *AttributeExpr) float64 {
	if a.Validation.ExclusiveMaximum != nil {
		return *a.Validation.ExclusiveMaximum
	}
	if a.Validation.Maximum != nil {
		return *a.Validation.Maximum
	}
	return math.Inf(1)
}

func readMinimum(a *AttributeExpr) float64 {
	if a.Validation.ExclusiveMinimum != nil {
		return *a.Validation.ExclusiveMinimum
	}
	if a.Validation.Minimum != nil {
		return *a.Validation.Minimum
	}
	return 0
}

func randomMinOnlyValue(kind Kind, r *ExampleGenerator, minimum float64, sign int) any {
	switch kind {
	case IntKind:
		return sign * (r.Int() + int(minimum))
	case Int32Kind:
		return int32(sign) * (r.Int32() + int32(minimum))
	case Int64Kind:
		return int64(sign) * (r.Int64() + int64(minimum))
	case UIntKind:
		return r.UInt() + uint(minimum)
	case UInt32Kind:
		return r.UInt32() + uint32(minimum)
	case UInt64Kind:
		return r.UInt64() + uint64(minimum)
	case Float32Kind:
		return float32(sign) * (r.Float32() + float32(minimum))
	default:
		return float64(sign) * (r.Float64() + minimum)
	}
}

func randomBoundedValue(kind Kind, r *ExampleGenerator, minimum, maximum float64) any {
	delta := maximum - minimum
	switch kind {
	case IntKind:
		return r.Int()%positiveIntDelta(delta) + int(minimum)
	case Int32Kind:
		return r.Int32()%int32(positiveIntDelta(delta)) + int32(minimum)
	case Int64Kind:
		return r.Int64()%int64(positiveIntDelta(delta)) + int64(minimum)
	case UIntKind:
		return r.UInt()%uint(positiveIntDelta(delta)) + uint(minimum)
	case UInt32Kind:
		return r.UInt32()%uint32(positiveIntDelta(delta)) + uint32(minimum)
	case UInt64Kind:
		return r.UInt64()%uint64(positiveIntDelta(delta)) + uint64(minimum)
	case Float32Kind:
		return r.Float32()*float32(delta) + float32(minimum)
	default:
		return r.Float64()*delta + minimum
	}
}

func positiveIntDelta(delta float64) int {
	if delta < 1 {
		return 1
	}
	return int(delta)
}

func minValueForKind(kind Kind, minimum float64) any {
	switch kind {
	case IntKind:
		return int(minimum)
	case Int32Kind:
		return int32(minimum)
	case Int64Kind:
		return int64(minimum)
	case UIntKind:
		return uint(minimum)
	case UInt32Kind:
		return uint32(minimum)
	case UInt64Kind:
		return uint64(minimum)
	case Float32Kind:
		return float32(minimum)
	default:
		return minimum
	}
}

func checkPattern(a *AttributeExpr, example any) bool {
	if !hasPatternValidation(a) {
		return true
	}
	pattern := a.Validation.Pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic("unreachable: invalid pattern '" + pattern + "' should have been caught by ValidationExpr.Validate")
	}
	if !re.MatchString(fmt.Sprint(example)) {
		return false
	}
	return true
}

func checkMinMaxValue(a *AttributeExpr, example any) bool {
	if !hasMinMaxValidation(a) {
		return true
	}
	var minimum *float64
	if a.Validation.ExclusiveMinimum != nil {
		minimum = a.Validation.ExclusiveMinimum
	}
	if a.Validation.Minimum != nil {
		minimum = a.Validation.Minimum
	}
	if minimum != nil {
		if v, ok := example.(int); ok && float64(v) < *minimum {
			return false
		} else if v, ok := example.(float64); ok && v < *minimum {
			return false
		}
	}
	var maximum *float64
	if a.Validation.ExclusiveMaximum != nil {
		maximum = a.Validation.ExclusiveMaximum
	}
	if a.Validation.Maximum != nil {
		maximum = a.Validation.Maximum
	}
	if maximum != nil {
		if v, ok := example.(int); ok && float64(v) > *maximum {
			return false
		} else if v, ok := example.(float64); ok && v > *maximum {
			return false
		}
	}
	return true
}
