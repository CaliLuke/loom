import ValueContract.MapKeys

namespace ValueContract.Candidate

variable {keys : KeyCodec}

def rawNil : Value → Bool
  | .host _ payload => rawNil payload
  | .null | .nilArray | .nilMap | .nilBytes => true
  | _ => false

def rawSlice : Value → Option (List Value)
  | .host _ payload => rawSlice payload
  | .array values => some values
  | .scalar (.bytes bytes) => some (bytes.map (fun byte => .scalar (.integer byte.toNat)))
  | _ => none

def rawObject : Value → Option (List (String × Value))
  | .host _ payload => rawObject payload
  | .object [] entries => some entries
  | .map entries => entries.mapM fun entry => match entry.1 with
    | .string name => some (name, entry.2)
    | _ => none
  | _ => none

/-- Existing finite raw-Any enum comparison, separate from declared collection
normalization and from JSON codec equality. Typed nil bytes remain distinguishable
from a non-nil empty byte slice, and Bytes never become authored String here. -/
def rawAnyLayer (recursive : Value → Value → Prop) (left right : Value) : Prop :=
  match left, right with
  | .host identity payload, .host other otherPayload =>
    identity = other ∨ recursive payload otherPayload
  | .host _ payload, right => recursive payload right
  | left, .host _ payload => recursive left payload
  | left, right =>
    if rawNil left || rawNil right then rawNil left = true ∧ rawNil right = true
    else match left, right with
    | .scalar l, .scalar r => scalarEqual l r = true
    | _, _ => match rawObject left, rawObject right with
      | some l, some r => l.length = r.length ∧ ∀ entry ∈ l, ∃ other ∈ r,
        entry.1 = other.1 ∧ recursive entry.2 other.2
      | _, _ => match rawSlice left, rawSlice right with
        | some l, some r => All₂ (recursive) l r
        | _, _ => False

def rawAnyEquivalentAt : Nat → Value → Value → Prop
  | 0, _, _ => False
  | depth + 1, left, right => rawAnyLayer (rawAnyEquivalentAt depth) left right

/-- Enum/default equality after the existing JSON collection normalization.
This relation is NOT general semantic or target-observation equality: only this
contract policy collapses nil containers to empty. It ignores entry order but
retains branch identities, null and absence. Duplicate entries are rejected by
complete typing of both the input and declaration enum members. -/
def enumEquivalentAt (keys : KeyCodec) : Nat → Value → Value → Prop
  | 0, _, _ => False
  | depth + 1, left, right =>
    match left, right with
    | .host identity payload, right => rawAnyLayer (rawAnyEquivalentAt depth) (.host identity payload) right
    | left, .host identity payload => rawAnyLayer (rawAnyEquivalentAt depth) left (.host identity payload)
    | .absent, .absent | .null, .null | .nilArray, .nilArray | .nilMap, .nilMap => True
    | .nilArray, .array [] | .array [], .nilArray => True
    | .nilMap, .map [] | .map [], .nilMap => True
    | .scalar l, .scalar r => scalarEqual l r = true
    | .array l, .array r => All₂ (enumEquivalentAt keys depth) l r
    | .object l le, .object r re =>
      l.length = r.length ∧
      (∀ entry ∈ l, ∃ other ∈ r, entry.1 = other.1 ∧ enumEquivalentAt keys depth entry.2 other.2) ∧
      le.length = re.length ∧
      (∀ entry ∈ le, ∃ other ∈ re, entry.1 = other.1 ∧ enumEquivalentAt keys depth entry.2 other.2)
    | .map l, .map r => l.length = r.length ∧
      ∀ entry ∈ l, ∃ other ∈ r,
        SameKey keys entry.1 other.1 ∧ enumEquivalentAt keys depth entry.2 other.2
    | .union identity branch payload, .union other otherBranch otherPayload =>
      identity = other ∧ branch = otherBranch ∧ enumEquivalentAt keys depth payload otherPayload
    | .any left, .any right => rawAnyEquivalentAt depth left right
    | _, _ => False

def EnumEquivalent (keys : KeyCodec) (left right : Value) : Prop := ∃ depth, enumEquivalentAt keys depth left right

def EnumAllows (keys : KeyCodec) (enumeration : Option (List Value)) (value : Value) : Prop :=
  match enumeration with
  | none => True
  | some values => ∃ member ∈ values, EnumEquivalent keys value member

/-- Extra open-object data has no declaration with which to reinterpret it.
It must itself be a finite built-in value. Raw cycles/custom values cannot
enter by pretending to be unconstrained object members. -/
def freeValueAt (keys : KeyCodec) : Nat → Value → Prop
  | 0, _ => False
  | depth + 1, value =>
    match value with
    | .null | .scalar _ | .nilArray | .nilMap | .nilBytes => True
    | .host _ payload => freeValueAt keys depth payload
    | .array items => All (freeValueAt keys depth) items
    | .object fields extra => fields = [] ∧ (extra.map Prod.fst).Nodup ∧
      ∀ entry ∈ extra, freeValueAt keys depth entry.2
    | .map entries => AdmissibleKeys keys (entries.map Prod.fst) ∧
      ∀ entry ∈ entries, freeValueAt keys depth entry.2
    | _ => False

def FreeValue (keys : KeyCodec) (value : Value) : Prop := ∃ depth, freeValueAt keys depth value

/-- Complete and partial typing differ only on missing required members.
Supplied constraints and selected branch identities are checked in both modes.
This independent judgment never calls a resolver or projector. -/
def typedAt (keys : KeyCodec) : Nat → Declarations → ExternalScalarChecks → Bool → Identity → Value → Prop
  | 0, _, _, _, _, _ => False
  | depth + 1, declarations, checks, complete, identity, value =>
    ∃ declaration ∈ declarations, declaration.identity = identity ∧
      EnumAllows keys declaration.enumeration value ∧
      match declaration.contract, value with
      | .scalar kind rules, .scalar scalar =>
        scalar.kind = kind ∧ scalarAllowed checks rules scalar = true
      | .nullable _, .null => True
      | .nullable child, value | .alias child, value =>
        typedAt keys depth declarations checks complete child value
      | .nonNull child, value => valueIsNull value = false ∧
        typedAt keys depth declarations checks complete child value
      | .any, .any payload => FreeValue keys payload
      | .array child bounds, .array values => lengthAllowed bounds values.length = true ∧
        All (typedAt keys depth declarations checks complete child) values
      | .array _ bounds, .nilArray => lengthAllowed bounds 0 = true
      | .map kind rules child bounds, .map entries =>
        lengthAllowed bounds entries.length = true ∧ AdmissibleKeys keys (entries.map Prod.fst) ∧
        ∀ entry ∈ entries, mapKeyCompatible kind entry.1 = true ∧
          scalarAllowed checks rules entry.1 = true ∧
          typedAt keys depth declarations checks complete child entry.2
      | .map _ _ _ bounds, .nilMap => lengthAllowed bounds 0 = true
      | .object members isOpen, .object fields extra =>
        (fields.map Prod.fst).Nodup ∧
        (∀ entry ∈ fields, ∃ member ∈ members, entry.1 = member.identity) ∧
        (∀ member ∈ members,
          (∃ value, (member.identity, value) ∈ fields ∧ value ≠ .absent ∧
            typedAt keys depth declarations checks complete member.child value) ∨
          ((complete = false ∨ member.required = false) ∧
            ∀ value, (member.identity, value) ∈ fields → value = .absent)) ∧
        (isOpen = true ∨ extra = []) ∧ (extra.map Prod.fst).Nodup ∧
        (∀ entry ∈ extra, ∀ member ∈ members,
          entry.1 ≠ member.sourceName ∧ entry.1 ≠ member.wireAlias) ∧
        (∀ entry ∈ extra, FreeValue keys entry.2)
      | .union occurrence alternatives, .union actual branch payload => actual = occurrence ∧
        ∃ alternative ∈ alternatives, alternative.identity = branch ∧
          typedAt keys depth declarations checks complete alternative.child payload
      | _, _ => False

def Typed (keys : KeyCodec) (declarations : Declarations) (checks : ExternalScalarChecks)
    (complete : Bool) (identity : Identity) (value : Value) : Prop :=
  ∃ depth, typedAt keys depth declarations checks complete identity value

/-- Every declared enum alternative must independently be a complete typed value.
This premise rules out malformed duplicate-bearing enum members; enum membership
alone cannot establish declaration validity. -/
def ValidEnumDeclarations (keys : KeyCodec) (declarations : Declarations) (checks : ExternalScalarChecks) : Prop :=
  ∀ declaration ∈ declarations, ∀ values, declaration.enumeration = some values →
    ∀ value ∈ values, Typed keys declarations checks true declaration.identity value

/-- Named expansion is allowed only when it decreases this rank without
consuming input. Object/collection recursion is deliberately excluded from
this edge set, so arbitrary finite recursive trees remain in the domain. -/
def nonconsumingChildren : Contract → List Identity
  | .nullable child | .nonNull child | .alias child => [child]
  | .union _ alternatives => alternatives.map Alternative.child
  | _ => []

def consumingChildren : Contract → List Identity
  | .array child _ | .map _ _ child _ => [child]
  | .object members _ => members.map Member.child
  | _ => []

def WellFormedDeclarations (declarations : Declarations) : Prop :=
  (declarations.map Declaration.identity).Nodup ∧
  ∀ declaration ∈ declarations,
    (∀ child ∈ nonconsumingChildren declaration.contract,
      ∃ target ∈ declarations, target.identity = child ∧
        target.expansionRank < declaration.expansionRank) ∧
    (∀ child ∈ consumingChildren declaration.contract,
      ∃ target ∈ declarations, target.identity = child) ∧
    match declaration.contract with
    | .object members _ =>
        (members.map Member.identity).Nodup ∧
        (members.map Member.sourceName).Nodup ∧
        (members.map Member.wireAlias).Nodup
    | .union _ alternatives => (alternatives.map Alternative.identity).Nodup
    | _ => True

end ValueContract.Candidate
