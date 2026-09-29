/-
Concrete authored primitive admission, before coercion. The table receives no
value, codec, role, target, graph, or selected branch. Concrete Go tag extraction
is a tested boundary; these proofs do not prove reflection or DSL extraction.
-/
import ValueContract.Model

namespace ValueContract.Candidate

/-- Raw Go primitive identity, separate from numeric meaning, literal formatting
and host equality class. NativeBytes denotes precisely []byte; named containers
and arrays instead carry sequence evidence. Normalized controls are abstract,
never evidence of a concrete Go type. -/
inductive RawPrimitive where
  | builtinBoolean | builtinString | nativeBytes
  | builtinInt | builtinInt8 | builtinInt16 | builtinInt32 | builtinInt64
  | builtinUInt | builtinUInt8 | builtinUInt16 | builtinUInt32 | builtinUInt64
  | builtinFloat32 | builtinFloat64
  | definedScalar (kind : ScalarKind)
  | normalized (kind : ScalarKind)
  deriving DecidableEq, Repr

/-- Int and Int64 are different source policies even on a 64-bit host.
The abstract policy is only for explicitly normalized model controls. -/
inductive SourcePrimitive where
  | boolean | string | bytes
  | int | int32 | int64 | uint | uint32 | uint64 | float32 | float64
  | any
  | abstract (kind : ScalarKind)
  deriving DecidableEq, Repr

/-- Semantic category is sufficient only for explicitly abstract admission. -/
def RawPrimitive.kind : RawPrimitive → ScalarKind
  | .builtinBoolean => .boolean
  | .builtinString => .string
  | .nativeBytes => .bytes
  | .builtinFloat32 | .builtinFloat64 => .decimal
  | .definedScalar kind | .normalized kind => kind
  | _ => .integer

/-- Executable table organized by raw concrete type, like IsCompatible. -/
def sourceAdmits (source : SourcePrimitive) (raw : RawPrimitive) : Bool :=
  match source with
  | .any => true
  | .abstract kind => raw.kind == kind ||
      (kind == .bytes && raw.kind == .string) ||
      (kind == .decimal && raw.kind == .integer)
  | _ => match raw with
    | .builtinBoolean => source == .boolean
    | .builtinString => source == .string || source == .bytes
    | .nativeBytes => source == .bytes
    | .builtinInt | .builtinInt8 | .builtinInt16 | .builtinInt32 |
        .builtinUInt | .builtinUInt8 | .builtinUInt16 | .builtinUInt32 =>
        source == .int || source == .int32 || source == .int64 ||
        source == .uint || source == .uint32 || source == .uint64 ||
        source == .float32 || source == .float64
    | .builtinInt64 | .builtinUInt64 =>
        source == .int64 || source == .uint64 || source == .float32 || source == .float64
    | .builtinFloat32 | .builtinFloat64 => source == .float32 || source == .float64
    | .definedScalar _ | .normalized _ => false

/-- Independent relation organized by declared source type. It does not call
sourceAdmits or assume its result. Constraints and coercion remain separate. -/
def SourceAdmits (source : SourcePrimitive) (raw : RawPrimitive) : Prop :=
  match source with
  | .any => True
  | .boolean => raw = .builtinBoolean
  | .string => raw = .builtinString
  | .bytes => raw = .builtinString ∨ raw = .nativeBytes
  | .int | .int32 | .uint | .uint32 =>
      raw ∈ [.builtinInt, .builtinInt8, .builtinInt16, .builtinInt32,
        .builtinUInt, .builtinUInt8, .builtinUInt16, .builtinUInt32]
  | .int64 | .uint64 =>
      raw ∈ [.builtinInt, .builtinInt8, .builtinInt16, .builtinInt32, .builtinInt64,
        .builtinUInt, .builtinUInt8, .builtinUInt16, .builtinUInt32, .builtinUInt64]
  | .float32 | .float64 =>
      raw ∈ [.builtinInt, .builtinInt8, .builtinInt16, .builtinInt32, .builtinInt64,
        .builtinUInt, .builtinUInt8, .builtinUInt16, .builtinUInt32, .builtinUInt64,
        .builtinFloat32, .builtinFloat64]
  | .abstract kind => raw.kind = kind ∨
      (kind = .bytes ∧ raw.kind = .string) ∨
      (kind = .decimal ∧ raw.kind = .integer)

/-- Full finite type-space correspondence, for positive and negative pairs. -/
theorem sourceAdmits_iff (source : SourcePrimitive) (raw : RawPrimitive) :
    sourceAdmits source raw = true ↔ SourceAdmits source raw := by
  cases source <;> cases raw <;>
    simp [sourceAdmits, SourceAdmits, RawPrimitive.kind, or_assoc]

theorem sourceAdmits_false_iff (source : SourcePrimitive) (raw : RawPrimitive) :
    sourceAdmits source raw = false ↔ ¬ SourceAdmits source raw := by
  rw [← sourceAdmits_iff]
  cases sourceAdmits source raw <;> simp

/-- Formatting methods and equal numeric meaning cannot manufacture admission. -/
theorem definedScalar_not_concrete {source : SourcePrimitive} (kind : ScalarKind)
    (notAny : source ≠ .any) (concrete : ∀ kind, source ≠ .abstract kind) :
    sourceAdmits source (.definedScalar kind) = false := by
  cases source <;> simp_all [sourceAdmits]

theorem normalized_not_concrete {source : SourcePrimitive} (kind : ScalarKind)
    (notAny : source ≠ .any) (concrete : ∀ kind, source ≠ .abstract kind) :
    sourceAdmits source (.normalized kind) = false := by
  cases source <;> simp_all [sourceAdmits]

/-- Source primitive restrictions do not narrow finite raw Any values. -/
theorem sourceAdmits_any (raw : RawPrimitive) : sourceAdmits .any raw = true := rfl

-- Representative host distinctions from the 321-case Go source matrix. The
-- universal iff above covers every tag pairing; collection traversal and empty
-- collections remain in the resolver rather than this primitive table.
theorem sourceAdmission_machine_int_is_not_int64 :
    sourceAdmits .int .builtinInt = true ∧
    sourceAdmits .int .builtinInt64 = false ∧
    sourceAdmits .int64 .builtinInt64 = true := by decide

theorem sourceAdmission_unsigned_width :
    sourceAdmits .int32 .builtinUInt32 = true ∧
    sourceAdmits .int32 .builtinUInt64 = false ∧
    sourceAdmits .uint64 .builtinInt64 = true := by decide

theorem sourceAdmission_numeric_cross_kind :
    sourceAdmits .float32 .builtinInt64 = true ∧
    sourceAdmits .float64 .builtinUInt64 = true ∧
    sourceAdmits .float32 .builtinFloat64 = true ∧
    sourceAdmits .int64 .builtinFloat32 = false := by decide

theorem sourceAdmission_bytes_text :
    sourceAdmits .bytes .nativeBytes = true ∧
    sourceAdmits .bytes .builtinString = true ∧
    sourceAdmits .string .nativeBytes = false ∧
    sourceAdmits .bytes (.definedScalar .bytes) = false := by decide

theorem sourceAdmission_named_scalars :
    sourceAdmits .boolean (.definedScalar .boolean) = false ∧
    sourceAdmits .string (.definedScalar .string) = false ∧
    sourceAdmits .int (.definedScalar .integer) = false ∧
    sourceAdmits .float64 (.definedScalar .decimal) = false := by decide

theorem sourceAdmission_abstract_is_explicit :
    sourceAdmits (.abstract .integer) (.normalized .integer) = true ∧
    sourceAdmits .int (.normalized .integer) = false ∧
    sourceAdmits (.abstract .bytes) (.normalized .string) = true ∧
    sourceAdmits (.abstract .decimal) (.normalized .integer) = true := by decide

/-- Complete finite raw tag domain, including all defined/normalized categories. -/
def allRawPrimitives : List RawPrimitive := [
  .builtinBoolean, .builtinString, .nativeBytes,
  .builtinInt, .builtinInt8, .builtinInt16, .builtinInt32, .builtinInt64,
  .builtinUInt, .builtinUInt8, .builtinUInt16, .builtinUInt32, .builtinUInt64,
  .builtinFloat32, .builtinFloat64,
  .definedScalar .boolean, .definedScalar .string, .definedScalar .bytes,
  .definedScalar .integer, .definedScalar .decimal,
  .normalized .boolean, .normalized .string, .normalized .bytes,
  .normalized .integer, .normalized .decimal]

/-- Every concrete declaration policy, with Any kept explicit. -/
def concreteSourcePrimitives : List SourcePrimitive := [
  .boolean, .string, .bytes, .int, .int32, .int64, .uint, .uint32, .uint64,
  .float32, .float64, .any]

theorem allRawPrimitives_complete (raw : RawPrimitive) : raw ∈ allRawPrimitives := by
  cases raw <;> simp [allRawPrimitives]
  all_goals rename_i kind; cases kind <;> simp

/-- Exhaustive 300-cell concrete table check: 105 accepted and 195 rejected
pairs, including every defined/normalized negative and all raw Any positives.
The independent universal iff above establishes each pair, not only this count. -/
theorem sourceAdmission_exhaustive_matrix :
    allRawPrimitives.length * concreteSourcePrimitives.length = 300 ∧
    ((concreteSourcePrimitives.flatMap fun source =>
      allRawPrimitives.map fun raw => sourceAdmits source raw).filter id).length = 105 ∧
    ((concreteSourcePrimitives.flatMap fun source =>
      allRawPrimitives.map fun raw => sourceAdmits source raw).filter (! ·)).length = 195 := by
  set_option maxRecDepth 4096 in decide

end ValueContract.Candidate
