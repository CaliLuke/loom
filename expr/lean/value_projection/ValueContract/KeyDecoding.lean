import ValueContract.MapKeys

namespace ValueContract.Candidate

/-- Key decoding is not the inverse of canonical spelling: accepted aliases must
remain visible when validating untagged alternatives. Built-in heterogeneous
keys materialize into the JSON string-key domain, after collision checking. -/
def decodeKey (codec : NumericCodec) (kind : MapKeyKind) (format : NumericFormat) (integers : IntegerFormat) (name : String) : Option Scalar :=
  match kind with
  | .builtin | .scalar .string => some (.string name)
  | .scalar .boolean =>
    if name = "true" then some (.boolean true)
    else if name = "false" then some (.boolean false) else none
  | .scalar .integer => (codec.decodeKeyInteger integers name).map Scalar.integer
  | .scalar .decimal => (codec.decodeKeyDecimal format name).map
      (fun number => .decimal number.value.coefficient number.value.exponent format number.negativeZero)
  | .scalar .bytes => none

inductive KeyDecodes (codec : NumericCodec) (format : NumericFormat) (integers : IntegerFormat) : MapKeyKind → String → Scalar → Prop where
  | builtin : KeyDecodes codec format integers .builtin name (.string name)
  | string : KeyDecodes codec format integers (.scalar .string) name (.string name)
  | trueKey : KeyDecodes codec format integers (.scalar .boolean) "true" (.boolean true)
  | falseKey : KeyDecodes codec format integers (.scalar .boolean) "false" (.boolean false)
  | integer (decoded : codec.decodeKeyInteger integers name = some integer) :
      KeyDecodes codec format integers (.scalar .integer) name (.integer integer)
  | decimal (decoded : codec.decodeKeyDecimal format name = some number) :
      KeyDecodes codec format integers (.scalar .decimal) name (.decimal number.value.coefficient number.value.exponent format number.negativeZero)

/-- Target observation preserves declared scalar keys but observes heterogeneous
built-in keys by their final JSON member names. It never hides a collision:
canonical construction separately requires all names to be distinct. -/
inductive ObservedKey (codec : NumericCodec) : MapKeyKind → Scalar → Scalar → Prop where
  | scalar (compatible : mapKeyCompatible (.scalar kind) value = true) :
      ObservedKey codec (.scalar kind) value value
  | builtin (spelling : KeySpelling codec value name) :
      ObservedKey codec .builtin value (.string name)

def observeKey (codec : NumericCodec) (kind : MapKeyKind) (value : Scalar) : Option Scalar :=
  match kind with
  | .scalar _ => if mapKeyCompatible kind value then some value else none
  | .builtin => (encodeKey codec value).map Scalar.string

theorem keyDecoder_progress {codec kind format integers name value} (decoded : KeyDecodes codec format integers kind name value) :
    decodeKey codec kind format integers name = some value := by
  cases decoded <;> simp [decodeKey, *]

theorem keyDecoder_sound {codec kind format integers name value} (decoded : decodeKey codec kind format integers name = some value) :
    KeyDecodes codec format integers kind name value := by
  cases kind with
  | builtin =>
    simp [decodeKey] at decoded
    subst value
    exact .builtin
  | scalar kind =>
    cases kind with
    | string =>
      simp [decodeKey] at decoded
      subst value
      exact .string
    | bytes => simp [decodeKey] at decoded
    | boolean =>
      by_cases positive : name = "true"
      · subst name
        simp [decodeKey] at decoded
        subst value
        exact .trueKey
      · by_cases negative : name = "false"
        · subst name
          simp [decodeKey] at decoded
          subst value
          exact .falseKey
        · simp [decodeKey, positive, negative] at decoded
    | integer =>
      simp [decodeKey, Option.map_eq_some_iff] at decoded
      obtain ⟨integer, parsed, same⟩ := decoded
      subst value
      exact .integer parsed
    | decimal =>
      simp [decodeKey, Option.map_eq_some_iff] at decoded
      obtain ⟨number, parsed, same⟩ := decoded
      subst value
      exact .decimal parsed

theorem keyDecoder_iff (codec : NumericCodec) (kind : MapKeyKind) (format : NumericFormat) (integers : IntegerFormat) (name : String) (value : Scalar) :
    decodeKey codec kind format integers name = some value ↔ KeyDecodes codec format integers kind name value :=
  ⟨keyDecoder_sound, keyDecoder_progress⟩

end ValueContract.Candidate
