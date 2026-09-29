import ValueContract.Observation

namespace ValueContract.Candidate

/-- The carrier used by a target proof. Documentation-only occurrences have no
runtime decoder; runtime-backed occurrences must satisfy both independent checks. -/
inductive TargetUse where
  | runtime
  | documentation
  deriving DecidableEq, Repr

/-- Optional field fragments retain multiplicity until duplicate names have been
checked. In particular, omission is not implemented by overwriting a map entry. -/
def retainedMembers (entries : List (String × Option Wire)) : List (String × Wire) :=
  entries.filterMap fun entry => entry.2.map (entry.1, ·)

/-- Independent canonical representation of an ALREADY OBSERVED target value.
It describes spelling, omission and envelopes, not validation or projection
success. Depth indexes finite derivations, not a fixed supported-value bound.
Whole-node enum/range constraints belong to schema and runtime acceptance. -/
def canonicalAt : Nat → ScalarCodecs → Targets → Identity → Value → Wire → Prop
  | 0, _, _, _, _, _ => False
  | depth + 1, codecs, targets, identity, value, wire =>
    ∃ declaration ∈ targets, declaration.identity = identity ∧
    match declaration.target, value, wire with
    | .scalar encoding kind _, .scalar scalar, wire =>
      scalar.kind = kind ∧ CanonicalScalar codecs encoding scalar wire
    | .nullable _, .null, .null => True
    | .nullable child, value, wire | .alias child, value, wire |
        .select _ child, value, wire => canonicalAt depth codecs targets child value wire
    | .nonNull child, value, wire => valueIsNull value = false ∧
      canonicalAt depth codecs targets child value wire
    | .array child _, .array values, .array wires =>
      All₂ (canonicalAt depth codecs targets child) values wires
    | .map _ _ child _, .map entries, .object wires =>
      ∃ fragments, All₂ (fun entry fragment =>
        KeySpelling codecs.numbers entry.1 fragment.1 ∧
        canonicalAt depth codecs targets child entry.2 fragment.2) entries fragments ∧
        (fragments.map Prod.fst).Nodup ∧ wires = orderedMembers fragments
    | .object members preserveAdditional, .object fields extra, .object wires =>
      ∃ fragments additional,
        All₂ (fun member fragment => fragment.1 = member.wireName ∧
          let childValue := memberValue fields member.identity
          match fragment.2 with
          | none => member.required = false ∧ (childValue = .absent ∨
              implicitDefaultOmitted member.presence childValue = true)
          | some childWire => childValue ≠ .absent ∧
              implicitDefaultOmitted member.presence childValue = false ∧
              canonicalAt depth codecs targets member.child childValue childWire)
          members fragments ∧
        (if preserveAdditional then All₂ (fun entry result => entry.1 = result.1 ∧
          match entry.2 with
          | .jsonSnapshot snapshot => snapshot = result.2 ∧ JSONValid codecs.numbers snapshot
          | _ => Materializes codecs entry.2 result.2) extra additional
        else additional = []) ∧
        let combined := retainedMembers fragments ++ additional
        (combined.map Prod.fst).Nodup ∧ wires = orderedMembers combined
    | .union occurrence style alternatives, .union actual branch payload, wire =>
      occurrence = actual ∧ ∃ alternative ∈ alternatives,
        alternative.identity = branch ∧ ∃ childWire,
          canonicalAt depth codecs targets alternative.child payload childWire ∧
          match style with
          | .untagged => wire = childWire
          | .tagged tagKey valueKey =>
              wire = .object (orderedMembers [(tagKey, .text alternative.wireName),
                (valueKey, childWire)])
          | .protobuf => wire = .oneof alternative.wireName childWire
    | .any, .jsonSnapshot snapshot, wire =>
      wire = snapshot ∧ JSONValid codecs.numbers snapshot
    | _, _, _ => False

def Canonical (codecs : ScalarCodecs) (targets : Targets) (identity : Identity)
    (value : Value) (wire : Wire) : Prop :=
  ∃ depth, canonicalAt depth codecs targets identity value wire

end ValueContract.Candidate
