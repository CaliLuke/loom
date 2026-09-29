import ValueContract.ScalarSemantics

namespace ValueContract.Candidate

/-- Uses the same numeric codec as scalar JSON representation. Its decimal
formatter models the production jsonkey.Name float path, not a second unrelated
numeric spelling rule. Correct destination precision remains a tested boundary. -/
abbrev KeyCodec := NumericCodec

def encodeKey (codec : KeyCodec) : Scalar → Option String
  | .boolean value => some (if value then "true" else "false")
  | .integer value _ => some (toString value)
  | .decimal coefficient exponent format negativeZero _ => some (codec.encodeDecimal coefficient exponent format negativeZero)
  | .string value => some value
  | .bytes _ => none

inductive KeySpelling (codec : KeyCodec) : Scalar → String → Prop where
  | boolean : KeySpelling codec (.boolean value) (if value then "true" else "false")
  | integer : KeySpelling codec (.integer value origin) (toString value)
  | decimal : KeySpelling codec (.decimal coefficient exponent format negativeZero origin)
      (codec.encodeDecimal coefficient exponent format negativeZero)
  | string : KeySpelling codec (.string value) value

theorem keySpelling_sound {codec key name} (encoded : encodeKey codec key = some name) :
    KeySpelling codec key name := by
  cases key <;> simp [encodeKey] at encoded <;> subst name <;> constructor

theorem keySpelling_progress {codec key name} (spelled : KeySpelling codec key name) :
    encodeKey codec key = some name := by
  cases spelled <;> rfl

theorem keySpelling_iff (codec : KeyCodec) (key : Scalar) (name : String) :
    encodeKey codec key = some name ↔ KeySpelling codec key name :=
  ⟨keySpelling_sound, keySpelling_progress⟩

/-- Declared-map identity follows the canonical member name. Numeric enum
equality and raw host equality are different contracts. Invalid keys never
compare equal merely because both encoders reject them. -/
def SameKey (keys : KeyCodec) (left right : Scalar) : Prop :=
  ∃ name, KeySpelling keys left name ∧ KeySpelling keys right name

def keyEqual (keys : KeyCodec) (left right : Scalar) : Bool :=
  match encodeKey keys left, encodeKey keys right with
  | some left, some right => left == right
  | _, _ => false

theorem keyEqual_iff (keys : KeyCodec) (left right : Scalar) :
    keyEqual keys left right = true ↔ SameKey keys left right := by
  constructor
  · intro equal
    cases l : encodeKey keys left <;> cases r : encodeKey keys right <;>
      simp only [keyEqual, l, r, Bool.false_eq_true, beq_iff_eq] at equal
    case some.some lname rname =>
      subst rname
      exact ⟨lname, keySpelling_sound l, keySpelling_sound r⟩
  · rintro ⟨name, leftName, rightName⟩
    simp [keyEqual, keySpelling_progress leftName, keySpelling_progress rightName]

/-- Key admissibility is stated through spelling evidence for the entire list,
not successful execution of a map encoder. -/
def AdmissibleKeys (keys : KeyCodec) (values : List Scalar) : Prop :=
  ∃ names, names.Nodup ∧ All₂ (KeySpelling keys) values names

/-- All spellings are obtained in an entry list BEFORE checking uniqueness.
No insertion or right-biased map conversion is permitted to hide a collision. -/
def nameKeys (codec : KeyCodec) (keys : List Scalar) : Except Failure (List String) := do
  let names ← keys.mapM fun key => match encodeKey codec key with
    | some name => .ok name
    | none => .error .invalid
  if names.Nodup then .ok names else .error .invalid

@[simp] private theorem bindOk (value : α) (next : α → Except ε β) :
    (Except.ok value >>= next) = next value := rfl
@[simp] private theorem bindError (error : ε) (next : α → Except ε β) :
    (Except.error error >>= next) = .error error := rfl
@[simp] private theorem mapOk (value : α) (next : α → β) :
    (next <$> (Except.ok value : Except ε α)) = .ok (next value) := rfl
@[simp] private theorem mapError (error : ε) (next : α → β) :
    (next <$> (Except.error error : Except ε α)) = .error error := rfl
@[simp] private theorem pureOk (value : α) : (pure value : Except ε α) = .ok value := rfl

private theorem pairwiseCons (relation : α → β → Prop) (a : α) (as : List α)
    (b : β) (bs : List β) :
    All₂ relation (a :: as) (b :: bs) ↔ relation a b ∧ All₂ relation as bs := by
  simp [All₂, and_left_comm]

private theorem encodedKeys_iff (codec : KeyCodec) (keys : List Scalar) (names : List String) :
    keys.mapM (fun key => match encodeKey codec key with
      | some name => (Except.ok name : Except Failure String)
      | none => .error .invalid) = .ok names ↔ All₂ (KeySpelling codec) keys names := by
  induction keys generalizing names with
  | nil => cases names <;> simp [All₂]
  | cons key rest ih =>
    cases encoded : encodeKey codec key with
    | none =>
      have impossible (name) : ¬ KeySpelling codec key name := by
        intro relation
        have progress := keySpelling_progress relation
        simp [encoded] at progress
      cases names <;> simp [List.mapM_cons, encoded, All₂, impossible]
    | some name =>
      have unique (other) : KeySpelling codec key other ↔ other = name := by
        constructor
        · intro relation
          have progress := keySpelling_progress relation
          simpa [encoded] using progress.symm
        · intro equal
          subst other
          exact keySpelling_sound encoded
      cases names with
      | nil =>
        cases restResult : rest.mapM (fun key => match encodeKey codec key with
          | some name => (Except.ok name : Except Failure String)
          | none => .error .invalid) <;>
          simp [List.mapM_cons, encoded, All₂, restResult]
      | cons head tail =>
        rw [pairwiseCons]
        simp only [List.mapM_cons, encoded]
        cases restResult : rest.mapM (fun key => match encodeKey codec key with
          | some name => (Except.ok name : Except Failure String)
          | none => .error .invalid) with
        | error failure =>
          have noTail := ih tail
          simp [restResult] at noTail
          simp [unique, noTail]
        | ok values =>
          have tailRelation := ih tail
          simp [restResult] at tailRelation
          simp [unique, ← tailRelation, eq_comm]

/-- Success preserves the full entry list and requires uniqueness only AFTER all
canonical spellings are known. This includes heterogeneous-key collisions. -/
theorem nameKeys_iff (codec : KeyCodec) (keys : List Scalar) (names : List String) :
    nameKeys codec keys = .ok names ↔ names.Nodup ∧ All₂ (KeySpelling codec) keys names := by
  unfold nameKeys
  cases encoded : keys.mapM (fun key => match encodeKey codec key with
    | some name => (Except.ok name : Except Failure String)
    | none => .error .invalid) with
  | error failure =>
    have noNames := encodedKeys_iff codec keys names
    simp [encoded] at noNames
    simp [noNames]
  | ok values =>
    have relation := encodedKeys_iff codec keys names
    simp [encoded] at relation
    by_cases unique : values.Nodup
    · simp [unique, ← relation]
      intro equal
      subst names
      exact unique
    · simp [unique, ← relation]
      intro distinct equal
      subst names
      exact unique distinct

theorem integerStringCollision (codec : KeyCodec) :
    nameKeys codec [.integer 1, .string "1"] = .error .invalid := by
  rfl

theorem integerStringCollisionReversed (codec : KeyCodec) :
    nameKeys codec [.string "1", .integer 1] = .error .invalid := by
  rfl

theorem booleanStringCollision (codec : KeyCodec) :
    nameKeys codec [.boolean true, .string "true"] = .error .invalid := by
  rfl

theorem booleanStringCollisionReversed (codec : KeyCodec) :
    nameKeys codec [.string "true", .boolean true] = .error .invalid := by
  rfl

theorem heterogeneousDistinctKeys (codec : KeyCodec) :
    nameKeys codec [.integer 1, .string "other", .boolean true] = .ok ["1", "other", "true"] := by
  rfl

theorem heterogeneousDistinctKeysReversed (codec : KeyCodec) :
    nameKeys codec [.boolean true, .string "other", .integer 1] = .ok ["true", "other", "1"] := by
  rfl

theorem bytesKeyRejected (codec : KeyCodec) (bytes : List UInt8) :
    nameKeys codec [.bytes bytes] = .error .invalid := by
  rfl

end ValueContract.Candidate
