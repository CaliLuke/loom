import ValueContract.NumericCoercionControls

namespace ValueContract.Candidate.SourceAdmissionControls
open ResolutionControls

/-- Concrete source policies keep machine Int separate from Int64 even when
both inputs have the same mathematical value and the same target encoding. -/
def machineInt : Declarations := [
  ⟨iid, 0, .scalar .integer {sourcePrimitive := some .int}, none⟩]

def wideInt : Declarations := [
  ⟨iid, 0, .scalar .integer {sourcePrimitive := some .int64}, none⟩]

theorem concreteAdmissionPrecedesNormalization (role : Role) :
    resolve machineInt checks numbers role iid (.scalar ⟨.integer 1, .builtinInt64⟩) =
      .error .invalid ∧
    resolve wideInt checks numbers role iid (.scalar ⟨.integer 1, .builtinInt64⟩) =
      .ok ⟨.scalar (.integer 1), []⟩ := by
  cases role <;> cbv

def float64 : Declarations := [
  ⟨iid, 0, .scalar .decimal {sourcePrimitive := some .float64, numericFormat := .binary64}, none⟩]

/-- A formatting callback which would normalize a named value to 2 cannot
make the original named Go scalar eligible for a concrete Float64 declaration. -/
theorem namedFormattingCannotAdmitSource (role : Role) :
    coerceScalar NumericCoercionControls.originNumbers .binary64 .decimal
      (.decimal 1 0 .binary64 false 1) = some (.decimal 2 0 .binary64) ∧
    resolve float64 checks NumericCoercionControls.originNumbers role iid
      (.scalar ⟨.decimal 1 0 .binary64 false 1, .definedScalar .decimal⟩) =
      .error .invalid := by
  cases role <;> cbv
  all_goals exact ⟨rfl, True.intro⟩

def keyed : Declarations := [
  ⟨sid, 0, .scalar .string {sourcePrimitive := some .string}, none⟩,
  ⟨oid, 0, .map (.scalar .integer) {sourcePrimitive := some .int} sid {}, none⟩]

theorem mapKeyAdmissionRetainsOriginalHost (role : Role) :
    resolve keyed checks numbers role oid
      (.map [(⟨.integer 1, .builtinInt64⟩, .scalar ⟨.string "v", .builtinString⟩)]) =
      .error .invalid ∧
    resolve keyed checks numbers role oid
      (.map [(⟨.integer 1, .builtinInt⟩, .scalar ⟨.string "v", .builtinString⟩)]) =
      .ok ⟨.map [(.integer 1, .scalar (.string "v"))], []⟩ := by
  cases role <;> cbv

/-- Authored object member names are builtin strings before map matching.
Object-to-map conversion must not replace that evidence with abstract tags. -/
theorem objectMapConversionPreservesStringAdmission (role : Role) :
    resolve [
      ⟨sid, 0, .scalar .string {sourcePrimitive := some .string}, none⟩,
      ⟨oid, 0, .map (.scalar .string) {sourcePrimitive := some .string} sid {}, none⟩]
      checks numbers role oid (.object [("key", .scalar ⟨.string "v", .builtinString⟩)]) =
      .ok ⟨.map [(.string "key", .scalar (.string "v"))], []⟩ := by
  cases role <;> cbv

theorem rawAnyDoesNotInheritConcreteAdmission (role : Role) :
    resolve [⟨iid, 0, .any, none⟩] checks numbers role iid
      (.host 17 (.scalar ⟨.integer 1 3, .definedScalar .integer⟩)) =
      .ok ⟨.any (.host 17 (.scalar (.integer 1 3))), []⟩ := by
  cases role <;> cbv

theorem abstractEvidenceCannotEnterConcreteSource (role : Role) :
    resolve machineInt checks numbers role iid (.scalar (.integer 1)) = .error .invalid := by
  cases role <;> cbv

end ValueContract.Candidate.SourceAdmissionControls
