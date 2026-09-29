import ValueContract.NumericRepresentationControls
import ValueContract.Decoding

namespace ValueContract.Candidate.MapIdentityControls
open NumericRepresentationControls

def root : Identity := ⟨30, 0⟩
def child : Identity := ⟨30, 1⟩

def declarations : Declarations := [
  ⟨root, 0, .map .builtin {} child {}, none⟩,
  ⟨child, 0, .scalar .string {}, none⟩]

def entries : List (Scalar × Value) := [
  (binaryTenth .binary32, .scalar (.string "narrow")),
  (binaryTenth .binary64, .scalar (.string "wide"))]

/-- Equal mathematical keys do not collide when their canonical names differ. -/
theorem equalMeaningKeysResolve (role : Role) :
    resolve declarations ResolutionControls.checks numbers role root
      (.map [(binaryTenth .binary32, .scalar (.string "narrow")),
        (binaryTenth .binary64, .scalar (.string "wide"))]) =
      .ok ⟨.map entries, []⟩ := by cases role <;> cbv

theorem mixedWidthKeysAreDistinct :
    keyEqual numbers (binaryTenth .binary32) (binaryTenth .binary64) = false ∧
    scalarEqual (binaryTenth .binary32) (binaryTenth .binary64) = true := by cbv

theorem swappedValuesDoNotMatchEnum :
    valueEqualAt numbers 4 true (.map entries)
      (.map [(binaryTenth .binary32, .scalar (.string "wide")),
        (binaryTenth .binary64, .scalar (.string "narrow"))]) = false := by cbv

/-- Across two separately valid maps, key kind does not change member identity. -/
theorem heterogeneousNamesMatchAcrossMaps :
    valueEqualAt numbers 3 true
      (.map [(.integer 1, .scalar (.string "value"))])
      (.map [(.string "1", .scalar (.string "value"))]) = true ∧
    valueEqualAt numbers 3 false
      (.map [(.integer 1, .scalar (.string "value"))])
      (.map [(.string "1", .scalar (.string "value"))]) = true := by cbv

theorem heterogeneousNamesCollideWithinMap :
    resolve declarations ResolutionControls.checks numbers .authoredExample root
      (.map [(.integer 1, .scalar (.string "first")),
        (.string "1", .scalar (.string "second"))]) = .error .invalid := by cbv

theorem unsupportedKeysNeverMatch :
    keyEqual numbers (.bytes [1]) (.bytes [1]) = false := by rfl

/-- Canonical names do not replace the raw Any host comparison contract. -/
theorem rawAnyDoesNotInheritDeclaredMapEquality :
    valueEqualAt numbers 8 true
      (.any (.host 1 (.map [(.integer 1, .scalar (.string "value"))])))
      (.any (.host 2 (.map [(.integer 1, .scalar (.string "value"))]))) = false := by cbv

def decoderNumbers : NumericCodec := {
  numbers with
  encodeInteger := toString
  decodeKeyInteger := fun _ text => if text = "0" ∨ text = "-0" then some 0 else none
  encodeDecimal := fun coefficient _ _ negativeZero =>
    if coefficient = 0 then (if negativeZero then "-0" else "0") else "1"
  decodeKeyDecimal := fun _ text =>
    if text = "1" ∨ text = "1.00000001" then some ⟨⟨1, 0⟩, false⟩
    else if text = "0" then some ⟨⟨0, 0⟩, false⟩
    else if text = "-0" then some ⟨⟨0, 0⟩, true⟩ else none
}

def decoderCodecs : ScalarCodecs := { codecs with numbers := decoderNumbers }

def targets (kind : ScalarKind) (rules : ScalarRules) : Targets := [
  {identity := root, expansionRank := 0, target := .map (.scalar kind) rules child {}},
  {identity := child, expansionRank := 0, target := .scalar .json .string {}}]

def twoKeys (first second : String) : Wire :=
  .object [(first, .text "first"), (second, .text "second")]

theorem decodedIntegerZeroAliasesCollide :
    decode decoderCodecs ResolutionControls.checks
      (targets .integer {integerFormat := .signed 64}) root (twoKeys "0" "-0") = .ok none := by cbv

theorem decodedRoundingCollisionRejected :
    decode decoderCodecs ResolutionControls.checks
      (targets .decimal {numericFormat := .binary32}) root
      (twoKeys "1" "1.00000001") = .ok none := by cbv

/-- Different wire member names can denote the same native decoded key. -/
theorem decodedSignedZeroCollisionRejected :
    decode decoderCodecs ResolutionControls.checks
      (targets .decimal {numericFormat := .binary32}) root (twoKeys "0" "-0") =
      .ok none := by cbv

end ValueContract.Candidate.MapIdentityControls
