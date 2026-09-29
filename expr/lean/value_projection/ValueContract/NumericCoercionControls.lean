import ValueContract.NumericRepresentationControls

namespace ValueContract.Candidate.NumericCoercionControls
open ResolutionControls

/-- Actual exact value of float64(0.1), distinct from widening float32(0.1). -/
def wideTenth : Scalar :=
  .decimal 1000000000000000055511151231257827021181583404541015625 (-55) .binary64

def narrowTenth : Scalar := NumericRepresentationControls.binaryTenth .binary32

def numbers : NumericCodec := {
  NumericRepresentationControls.numbers with
  literalDecimal := fun _ _ format _ _ =>
    if format = .binary32 then "0.1" else "0.10000000149011612"
  parseLiteral := fun format text =>
    if text = "0.1" ∨ text = "0.10000000149011612" then
      if format = .binary32 then
        some ⟨⟨100000001490116119384765625, -27⟩, false⟩
      else if format = .binary64 ∧ text = "0.1" then
        some ⟨⟨1000000000000000055511151231257827021181583404541015625, -55⟩, false⟩
      else none
    else none
}

def declared (format : NumericFormat) (bounds : NumericBounds := {}) : Declarations :=
  [⟨iid, 0, .scalar .decimal {numericFormat := format, numeric := bounds}, none⟩]

/-- Retained old exact-only coercion does not implement destination precision. -/
theorem exactOnlyCoercionMissesNormalization :
    coerceBuiltinScalar .decimal narrowTenth = some narrowTenth ∧
    scalarEqual narrowTenth wideTenth = false := by decide

theorem declaredWideUsesLiteral :
    coerceScalar numbers .binary64 .decimal narrowTenth = some wideTenth := by decide

/-- Role controls completeness, not whether floating-point normalization runs. -/
theorem allRolesUseDeclaredPrecision (role : Role) :
    resolve (declared .binary64) checks numbers role iid (.scalar narrowTenth) =
      .ok ⟨.scalar wideTenth, []⟩ := by
  cases role <;> cbv

def lowerBound : NumericBounds := {minimum := some ⟨1000000005, -10⟩}

theorem constraintsFollowNormalization :
    scalarAllowed checks {numeric := lowerBound} narrowTenth = true ∧
    resolve (declared .binary64 lowerBound) checks numbers .authoredExample iid
      (.scalar narrowTenth) = .error .invalid := by cbv

def alternatives : Declarations :=
  declared .binary64 lowerBound ++ [
    ⟨bid, 0, .scalar .decimal {numericFormat := .binary32, numeric := lowerBound}, none⟩,
    ⟨uid, 1, .union uid [⟨branchA, "narrow", bid⟩, ⟨branchB, "wide", iid⟩], none⟩]

theorem branchSelectionFollowsNormalization :
    resolve alternatives checks numbers .authoredExample uid (.scalar narrowTenth) =
      .ok ⟨.union uid branchA (.scalar narrowTenth), []⟩ := by cbv

def mapDeclarations : Declarations := [
  ⟨sid, 0, .scalar .string {}, none⟩,
  ⟨oid, 0, .map (.scalar .decimal) {numericFormat := .binary32} sid {}, none⟩]

theorem declaredPrecisionCollisionRejected :
    resolve mapDeclarations checks numbers .authoredExample oid
      (.map [(narrowTenth, .scalar (.string "first")),
        (NumericRepresentationControls.binaryTenth .binary64, .scalar (.string "second"))]) =
      .error .invalid := by cbv

/-- Numeric formatting methods are a source-literal boundary, independent of
JSON encoding and mathematical numeric equality. -/
def originNumbers : NumericCodec := {
  numbers with
  literalInteger := fun value origin => if origin = 2 then "3" else toString value
  literalDecimal := fun coefficient _ _ _ origin => if origin = 1 then "2" else toString coefficient
  parseLiteral := fun _ text => text.toInt?.map fun value => ⟨⟨value, 0⟩, false⟩
}

theorem literalOriginsRemainNumericallyEqual :
    scalarEqual (.decimal 1 0 .binary64 false 0) (.decimal 1 0 .binary64 false 1) = true ∧
    scalarEqual (.integer 1 0) (.integer 1 2) = true := by decide

theorem literalOriginsNormalizeIndependently :
    coerceScalar originNumbers .binary64 .decimal (.decimal 1 0 .binary64 false 0) =
      some (.decimal 1 0 .binary64) ∧
    coerceScalar originNumbers .binary64 .decimal (.decimal 1 0 .binary64 false 1) =
      some (.decimal 2 0 .binary64) ∧
    coerceScalar originNumbers .binary64 .decimal (.integer 1 2) =
      some (.decimal 3 0 .binary64) := by
  cbv
  exact ⟨rfl, rfl, rfl⟩

theorem literalOriginDoesNotControlJSON (codecs : ScalarCodecs) (encoding : Encoding)
    (coefficient exponent : Int) (format : NumericFormat) (negativeZero : Bool) (origin : Nat) :
    encodeScalar codecs encoding (.decimal coefficient exponent format negativeZero origin) =
      encodeScalar codecs encoding (.decimal coefficient exponent format negativeZero 0) := by rfl

def originDeclarations : Declarations := declared .binary64 ++ [
  ⟨aid, 0, .array iid {}, none⟩]

theorem sameValueDistinctLiteralOrigins (role : Role) :
    resolve originDeclarations checks originNumbers role aid
      (.array [.scalar (.decimal 1 0 .binary64 false 0),
        .scalar (.decimal 1 0 .binary64 false 1), .scalar (.integer 1 2)]) =
      .ok ⟨.array [.scalar (.decimal 1 0 .binary64), .scalar (.decimal 2 0 .binary64),
        .scalar (.decimal 3 0 .binary64)], []⟩ := by
  cases role <;> cbv

end ValueContract.Candidate.NumericCoercionControls
