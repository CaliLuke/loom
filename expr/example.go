package expr

import (
	"fmt"
	"math"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"
	"unicode"

	"github.com/CaliLuke/loom/internal/exactnum"
	loom "github.com/CaliLuke/loom/pkg"
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
	effective := effectiveExampleAttribute(a, r)
	if effective == nil {
		return nil
	}

	// enum should dominate, because the potential "examples" are fixed
	if hasEnumValidation(effective) {
		return byEnum(effective, r)
	}
	return effective.generatedExample(r)
}

func effectiveExampleAttribute(attribute *AttributeExpr, generator *ExampleGenerator) *AttributeExpr {
	constraints, prepared := generator.effectiveConstraints[attribute]
	_, named := attribute.Type.(UserType)
	if !prepared && !named && !hasEnumValidation(attribute) {
		return attribute
	}
	if !prepared {
		resolved, err := EffectiveConstraintsFor(attribute)
		if err != nil {
			return nil
		}
		constraints = &resolved
	}
	effective := *attribute
	effective.Validation = constraints.Validation().Lowered()
	if candidates, present := constraints.EnumCandidates(); present {
		effective.Validation.Values = candidates
		effective.Validation.EnumClauses = nil
	}
	return &effective
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
		if !checkLength(a, example) || !checkPattern(a, example) || !checkFormat(a, example) || !checkMinMaxValue(a, example) {
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
	return a.Validation != nil && len(a.Validation.Enums()) > 0
}

func hasFormatValidation(a *AttributeExpr) bool {
	return a.Validation != nil && len(a.Validation.Formats()) > 0
}

func hasPatternValidation(a *AttributeExpr) bool {
	return a.Validation != nil && len(a.Validation.Patterns()) > 0
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
	format := a.Validation.Formats()[0]
	if res, ok := map[ValidationFormat]any{
		FormatEmail:        r.Email(),
		FormatHostname:     r.Hostname(),
		FormatDate:         time.Unix(int64(r.Int())%1454957045, 0).UTC().Format(time.DateOnly), // to obtain a "fixed" rand
		FormatDateTime:     time.Unix(int64(r.Int())%1454957045, 0).UTC().Format(time.RFC3339),  // to obtain a "fixed" rand
		FormatIPv4:         r.IPv4Address().String(),
		FormatIPv6:         r.IPv6Address().String(),
		FormatIP:           r.IPv4Address().String(),
		FormatURI:          r.URL(),
		FormatURIReference: "/example",
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
// The first pattern proposes the candidate; generatedExample checks every
// conjoined pattern before publishing it.
func byPattern(a *AttributeExpr, r *ExampleGenerator) any {
	if !hasPatternValidation(a) {
		return false
	}
	pattern := a.Validation.Patterns()[0]
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
	kind := unalias(a.Type).Kind()
	domain := numericDomain(kind)
	low, high := domain.Limits()
	lowerPresent, upperPresent := false, false
	for _, endpoint := range []struct {
		bound            *NumericBound
		lower, exclusive bool
	}{
		{a.Validation.Minimum, true, false}, {a.Validation.ExclusiveMinimum, true, true},
		{a.Validation.Maximum, false, false}, {a.Validation.ExclusiveMaximum, false, true},
	} {
		if endpoint.bound == nil {
			continue
		}
		projected := ProjectNumericBound(kind, *endpoint.bound, endpoint.lower, endpoint.exclusive)
		if projected.Empty {
			return nil
		}
		value, ok := exactnum.FromScalar(projected.Value)
		if !ok {
			panic("invalid numeric projection")
		}
		if endpoint.lower {
			current, ok := exactnum.FromScalar(low)
			if !ok {
				panic("invalid numeric domain")
			}
			if value.Cmp(current) > 0 {
				low = projected.Value
			}
			lowerPresent = true
		} else {
			current, ok := exactnum.FromScalar(high)
			if !ok {
				panic("invalid numeric domain")
			}
			if value.Cmp(current) < 0 {
				high = projected.Value
			}
			upperPresent = true
		}
	}
	lower, lowOK := exactnum.FromScalar(low)
	upper, highOK := exactnum.FromScalar(high)
	if !lowOK || !highOK {
		panic("invalid numeric example interval")
	}
	if lower.Cmp(upper) > 0 {
		return nil
	}
	if !domain.Float {
		return sampleNumericInteger(kind, domain.Unsigned, low, high, r)
	}
	return sampleNumericFloat(domain.Bits, low, high, lowerPresent, upperPresent, r)
}

func sampleNumericFloat(bits int, low, high any, lowerPresent, upperPresent bool, r *ExampleGenerator) any {
	var minimum, maximum float64
	if bits == 32 {
		minimum, maximum = float64(low.(float32)), float64(high.(float32))
	} else {
		minimum, maximum = low.(float64), high.(float64)
	}
	fraction := min(1.0, max(0.0, r.Float64()))
	value := minimum*(1-fraction) + maximum*fraction
	if !upperPresent {
		value = minimum + fraction
	} else if !lowerPresent {
		value = maximum - fraction
	}
	value = min(maximum, max(minimum, value))
	if bits == 32 {
		return float32(value)
	}
	return value
}

func sampleNumericInteger(kind Kind, unsigned bool, low, high any, r *ExampleGenerator) any {
	var first, last uint64
	if unsigned {
		first, last = low.(uint64), high.(uint64)
	} else {
		first, last = uint64(low.(int64)), uint64(high.(int64))
	}
	offset := r.UInt64()
	span := last - first
	if span != math.MaxUint64 {
		offset %= span + 1
	}
	value := first + offset
	switch kind {
	case IntKind:
		return int(int64(value))
	case Int32Kind:
		return int32(int64(value))
	case Int64Kind:
		return int64(value)
	case UIntKind:
		return uint(value)
	case UInt32Kind:
		return uint32(value)
	default:
		return value
	}
}

func checkPattern(a *AttributeExpr, example any) bool {
	if !hasPatternValidation(a) {
		return true
	}
	for _, pattern := range a.Validation.Patterns() {
		re, err := regexp.Compile(pattern)
		if err != nil {
			panic("unreachable: invalid pattern '" + pattern + "' should have been caught by ValidationExpr.Validate")
		}
		if !re.MatchString(fmt.Sprint(example)) {
			return false
		}
	}
	return true
}

func checkFormat(a *AttributeExpr, example any) bool {
	if !hasFormatValidation(a) {
		return true
	}
	text, ok := example.(string)
	if !ok {
		return false
	}
	for _, format := range a.Validation.Formats() {
		if loom.ValidateFormat("example", text, loom.Format(format)) != nil {
			return false
		}
	}
	return true
}

func checkMinMaxValue(a *AttributeExpr, example any) bool {
	if a == nil {
		return true
	}
	return checkValidationMinMaxValue(a.Validation, example)
}

func checkValidationMinMaxValue(validation *ValidationExpr, example any) bool {
	if validation == nil || (validation.ExclusiveMinimum == nil && validation.Minimum == nil &&
		validation.ExclusiveMaximum == nil && validation.Maximum == nil) {
		return true
	}
	value, ok := exactnum.FromScalar(example)
	if !ok {
		return true
	}
	if minimum := validation.ExclusiveMinimum; minimum != nil {
		if value.Cmp(*minimum) <= 0 {
			return false
		}
	}
	if minimum := validation.Minimum; minimum != nil {
		if value.Cmp(*minimum) < 0 {
			return false
		}
	}
	if maximum := validation.ExclusiveMaximum; maximum != nil {
		if value.Cmp(*maximum) >= 0 {
			return false
		}
	}
	if maximum := validation.Maximum; maximum != nil {
		if value.Cmp(*maximum) > 0 {
			return false
		}
	}
	return true
}
