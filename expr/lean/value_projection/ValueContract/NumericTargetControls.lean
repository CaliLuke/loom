import ValueContract.CandidateControls
import ValueContract.Decoding

namespace ValueContract.Candidate.NumericTargetControls

/-- Small boundary table, independently exercised against Go by the target
conformance corpus. This is a routing specimen, not a proof of IEEE parsing. -/
def integerReading (format : IntegerFormat) (text : String) : Option Int :=
  match format, text with
  | .signed 64, "2147483648" => some 2147483648
  | .unsigned 64, "9223372036854775808" => some 9223372036854775808
  | .signed _, "-0" => some 0
  | _, _ => none

def numbers : NumericCodec := {
  Controls.numbers with
  schemaNumber := fun text =>
    if text = "2147483648" then some ⟨2147483648, 0⟩
    else if text = "9223372036854775808" then some ⟨9223372036854775808, 0⟩
    else if text = "1e39" then some ⟨1, 39⟩
    else if text = "-0" then some ⟨0, 0⟩ else none
  decodeInteger := integerReading
  decodeKeyInteger := integerReading
  decodeDecimal := fun format text =>
    if text = "1e39" ∧ format = .binary64 then
      some ⟨⟨999999999999999939709166371603178586112, 0⟩, false⟩
    else if text = "-0" then some ⟨⟨0, 0⟩, true⟩ else none
  decodeKeyDecimal := fun format text =>
    if text = "-0" then some ⟨⟨0, 0⟩, true⟩
    else if text = "1e39" ∧ format = .binary64 then
      some ⟨⟨999999999999999939709166371603178586112, 0⟩, false⟩
    else none
}

def codecs : ScalarCodecs := { Controls.codecs with numbers := numbers }

theorem integerWidthChangesAcceptance :
    decodeScalar codecs .json .integer .exact (.signed 32) (.number "2147483648") = none ∧
    decodeScalar codecs .json .integer .exact (.signed 64) (.number "2147483648") =
      some (.integer 2147483648) := by exact ⟨rfl, rfl⟩

theorem integerSignednessChangesAcceptance :
    decodeScalar codecs .json .integer .exact (.signed 64) (.number "9223372036854775808") = none ∧
    decodeScalar codecs .json .integer .exact (.unsigned 64) (.number "9223372036854775808") =
      some (.integer 9223372036854775808) := by exact ⟨rfl, rfl⟩

/-- Mathematical range checks cannot distinguish these two parses of zero. -/
theorem signedZeroIsLexicalPolicy :
    decodeScalar codecs .json .integer .exact (.signed 64) (.number "-0") =
      some (.integer 0) ∧
    decodeScalar codecs .json .integer .exact (.unsigned 64) (.number "-0") = none := by
  exact ⟨rfl, rfl⟩

/-- The native JSON map decoder rejects the non-JSON alias that a standalone
signed-integer parser accepts. Parser-library success is not codec admission. -/
theorem nonJSONMemberAliasesRejected :
    decodeKey numbers (.scalar .integer) .exact (.signed 64) "+1" = none ∧
    decodeKey numbers (.scalar .integer) .exact (.unsigned 64) "+1" = none ∧
    decodeScalar codecs .json .integer .exact (.signed 64) (.number "+1") = none := by
  exact ⟨rfl, rfl, rfl⟩

theorem floatWidthChangesAcceptance :
    decodeScalar codecs .json .decimal .binary32 .mathematical (.number "1e39") = none ∧
    decodeScalar codecs .json .decimal .binary64 .mathematical (.number "1e39") =
      some (.decimal 999999999999999939709166371603178586112 0 .binary64 false) := by cbv

theorem floatResultRetainsTargetAndSign :
    decodeScalar codecs .json .decimal .binary32 .mathematical (.number "-0") =
      some (.decimal 0 0 .binary32 true) ∧
    decodeKey numbers (.scalar .decimal) .binary64 .mathematical "-0" =
      some (.decimal 0 0 .binary64 true) := by cbv

def root : Identity := ⟨20, 0⟩
def narrow : Identity := ⟨20, 1⟩
def wide : Identity := ⟨20, 2⟩
def narrowBranch : Identity := ⟨20, 3⟩
def wideBranch : Identity := ⟨20, 4⟩

def targets : Targets := [
  { identity := root, expansionRank := 1,
    target := .union root .untagged [⟨narrowBranch, "narrow", narrow⟩,
      ⟨wideBranch, "wide", wide⟩] },
  { identity := narrow, expansionRank := 0,
    target := .scalar .json .integer { integerFormat := .signed 32 } },
  { identity := wide, expansionRank := 0,
    target := .scalar .json .integer { integerFormat := .signed 64 } }]

/-- Both branches see exactly the same wire. The target graph supplies each
parser policy; there is no source kind or chosen source branch in this call. -/
theorem sameWireUsesEveryTargetPolicy :
    decode codecs Controls.noExternal targets root (.number "2147483648") =
      .ok (some (.union root wideBranch (.scalar (.integer 2147483648)))) := by cbv

/-- A runtime-unique branch does not establish schema oneOf uniqueness. -/
theorem schemaDoesNotBorrowIntegerParser :
    schema codecs Controls.noExternal targets narrow (.number "2147483648") = .ok true ∧
    schema codecs Controls.noExternal targets wide (.number "2147483648") = .ok true ∧
    schema codecs Controls.noExternal targets root (.number "2147483648") = .ok false := by cbv

/-- Retained old-policy counterexample: using one generic signed-64 parser for
both destinations admits the narrow branch and destroys runtime uniqueness. -/
theorem erasedPolicyLosesUniqueBranch :
    let erased : ScalarCodecs := { codecs with numbers :=
      { numbers with decodeInteger := fun _ => integerReading (.signed 64) } }
    decode erased Controls.noExternal targets root (.number "2147483648") = .ok none := by cbv

end ValueContract.Candidate.NumericTargetControls
