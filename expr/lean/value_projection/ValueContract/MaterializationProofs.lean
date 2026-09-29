import ValueContract.Materialization

namespace ValueContract.Candidate

/-- A finite collection of derivations has a common upper depth. This bounds a
particular proof tree, never the admitted input domain or executable traversal. -/
private theorem commonDepth {α : Type u} (P : Nat → α → Prop)
    (mono : ∀ {low high value}, low ≤ high → P low value → P high value)
    (items : List α) (each : ∀ value ∈ items, ∃ depth, P depth value) :
    ∃ depth, ∀ value ∈ items, P depth value := by
  induction items with
  | nil => exact ⟨0, by simp⟩
  | cons head tail ih =>
    obtain ⟨headDepth, headProof⟩ := each head (by simp)
    obtain ⟨tailDepth, tailProof⟩ := ih (fun value member => each value (by simp [member]))
    refine ⟨max headDepth tailDepth, ?_⟩
    intro value member
    rcases List.mem_cons.mp member with same | inside
    · subst value
      exact mono (Nat.le_max_left ..) headProof
    · exact mono (Nat.le_max_right ..) (tailProof value inside)

/-- More derivation depth preserves independent JSON validity. -/
theorem jsonValidAt_mono {numbers : NumericCodec} {low high : Nat} {wire : Wire}
    (increase : low ≤ high) (valid : jsonValidAt low numbers wire) :
    jsonValidAt high numbers wire := by
  induction low generalizing high wire with
  | zero => simp [jsonValidAt] at valid
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases wire <;> simp only [jsonValidAt] at valid ⊢
      all_goals try assumption
      · intro item member
        exact ih smaller (valid item member)
      · exact ⟨valid.1, fun entry member => ih smaller (valid.2 entry member)⟩

private theorem jsonValid_array_iff {numbers : NumericCodec} {items : List Wire} :
    JSONValid numbers (.array items) ↔ ∀ item ∈ items, JSONValid numbers item := by
  constructor
  · rintro ⟨depth, valid⟩
    cases depth with
    | zero => simp [jsonValidAt] at valid
    | succ depth => exact fun item member => ⟨depth, valid item member⟩
  · intro each
    obtain ⟨depth, all⟩ := commonDepth (fun depth value => jsonValidAt depth numbers value)
      (fun increase valid => jsonValidAt_mono increase valid) items each
    exact ⟨depth + 1, all⟩

private theorem jsonValid_object_iff {numbers : NumericCodec}
    {entries : List (String × Wire)} :
    JSONValid numbers (.object entries) ↔ (entries.map Prod.fst).Nodup ∧
      ∀ entry ∈ entries, JSONValid numbers entry.2 := by
  constructor
  · rintro ⟨depth, valid⟩
    cases depth with
    | zero => simp [jsonValidAt] at valid
    | succ depth => exact ⟨valid.1, fun entry member => ⟨depth, valid.2 entry member⟩⟩
  · rintro ⟨unique, each⟩
    obtain ⟨depth, all⟩ := commonDepth (fun depth (entry : String × Wire) => jsonValidAt depth numbers entry.2)
      (fun increase valid => jsonValidAt_mono increase valid) entries each
    exact ⟨depth + 1, unique, all⟩

/-- The executable checker exactly implements independent JSON validity for
arbitrary finite wire trees, including duplicate-key and numeric-lexeme checks. -/
theorem jsonValid_iff_JSONValid (numbers : NumericCodec) (wire : Wire) :
    jsonValid numbers wire = true ↔ JSONValid numbers wire := by
  cases wire with
  | null | boolean | text => simp [jsonValid, JSONValid]; exact ⟨1, by trivial⟩
  | number text =>
    simp only [jsonValid, JSONValid]
    constructor
    · intro valid; exact ⟨1, valid⟩
    · rintro ⟨depth, valid⟩
      cases depth <;> simp_all [jsonValidAt]
  | integer | decimal | bytes | oneof =>
    simp only [jsonValid, Bool.false_eq_true, JSONValid, false_iff, not_exists]
    intro depth
    cases depth <;> simp [jsonValidAt]
  | array items =>
    rw [jsonValid_array_iff]
    simp only [jsonValid, List.all_eq_true, List.mem_attach, true_implies, Subtype.forall]
    exact forall_congr' (fun item => forall_congr' (fun member =>
      jsonValid_iff_JSONValid numbers item))
  | object entries =>
    rw [jsonValid_object_iff]
    simp only [jsonValid, Bool.and_eq_true, decide_eq_true_eq,
      List.all_eq_true, List.mem_attach, true_implies, Subtype.forall]
    exact and_congr_right (fun _ => forall_congr' (fun entry => forall_congr' (fun member =>
      jsonValid_iff_JSONValid numbers entry.2)))
termination_by sizeOf wire
decreasing_by
  · have smaller := List.sizeOf_lt_of_mem member
    simp only [Wire.array.sizeOf_spec]
    omega
  · rcases entry with ⟨name, child⟩
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Wire.object.sizeOf_spec]
    simp only [Prod.mk.sizeOf_spec] at smaller
    omega

private theorem all₂_cons_iff {R : α → β → Prop} {a : α} {b : β}
    {left : List α} {right : List β} :
    All₂ R (a :: left) (b :: right) ↔ R a b ∧ All₂ R left right := by
  simp [All₂, and_left_comm]

private theorem all₂_mono {R S : α → β → Prop} {left : List α} {right : List β}
    (implies : ∀ a b, R a b → S a b) (related : All₂ R left right) : All₂ S left right :=
  ⟨related.1, fun pair member => implies pair.1 pair.2 (related.2 pair member)⟩

private theorem except_bind_ok {input : Except ε α} {next : α → Except ε β} {result : β} :
    (input >>= next) = .ok result ↔ ∃ value, input = .ok value ∧ next value = .ok result := by
  cases input <;> simp [Bind.bind, Except.bind]

private theorem mapM_ok_iff (f : α → Except ε β) (left : List α) (right : List β) :
    left.mapM f = .ok right ↔ All₂ (fun a b => f a = .ok b) left right := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂, pure, Except.pure]
  | cons head tail ih =>
    simp only [List.mapM_cons, except_bind_ok]
    cases right with
    | nil => simp [All₂, pure, Except.pure]
    | cons result rest =>
      simp only [all₂_cons_iff]
      constructor
      · rintro ⟨value, first, values, remaining, same⟩
        cases same
        exact ⟨first, (ih _).mp remaining⟩
      · rintro ⟨first, remaining⟩
        exact ⟨result, first, rest, (ih _).mpr remaining, rfl⟩

private theorem attach_mapM_eq (f : α → Except ε β) (items : List α) :
    items.attach.mapM (fun item => f item.val) = items.mapM f := by
  change items.attach.mapM (f ∘ Subtype.val) = _
  rw [← List.mapM_map]
  simp only [List.attach_map_subtype_val]

/-- Independent materialization derivations may be enlarged without changing
source values, member order, collision checks, or the selected wire result. -/
theorem materializesAt_mono {codecs : ScalarCodecs} {low high : Nat} {value : Value} {wire : Wire}
    (increase : low ≤ high) (valid : materializesAt low codecs value wire) :
    materializesAt high codecs value wire := by
  induction low generalizing high value wire with
  | zero => simp [materializesAt] at valid
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases value <;> cases wire <;> simp only [materializesAt] at valid ⊢
      all_goals try contradiction
      all_goals try assumption
      all_goals try exact ih smaller valid
      case nilArray.array items => cases items <;> simp_all
      case nilMap.object entries => cases entries <;> simp_all
      case array.array values results =>
        exact all₂_mono (fun _ _ relation => ih smaller relation) valid
      case object.object fields extra results =>
        cases fields with
        | cons => simp at valid
        | nil =>
          obtain ⟨unsorted, related, unique, same⟩ := valid
          exact ⟨unsorted, all₂_mono (fun _ _ relation => ⟨relation.1, ih smaller relation.2⟩)
            related, unique, same⟩
      case map.object entries results =>
        obtain ⟨unsorted, related, unique, same⟩ := valid
        exact ⟨unsorted, all₂_mono (fun _ _ relation => ⟨relation.1, ih smaller relation.2⟩)
          related, unique, same⟩

private theorem all₂_uniform {P : Nat → α → β → Prop}
    (mono : ∀ {low high a b}, low ≤ high → P low a b → P high a b)
    {left : List α} {right : List β} :
    All₂ (fun a b => ∃ depth, P depth a b) left right ↔
      ∃ depth, All₂ (P depth) left right := by
  constructor
  · intro related
    obtain ⟨depth, all⟩ := commonDepth (fun depth (pair : α × β) => P depth pair.1 pair.2)
      (fun increase valid => mono increase valid) (left.zip right) related.2
    exact ⟨depth, related.1, all⟩
  · rintro ⟨depth, related⟩
    exact all₂_mono (fun _ _ proof => ⟨depth, proof⟩) related

private theorem materializes_succ {codecs : ScalarCodecs} {value : Value} {wire : Wire} :
    Materializes codecs value wire ↔ ∃ depth, materializesAt (depth + 1) codecs value wire := by
  constructor
  · rintro ⟨depth, valid⟩
    cases depth with
    | zero => simp [materializesAt] at valid
    | succ depth => exact ⟨depth, valid⟩
  · rintro ⟨depth, valid⟩; exact ⟨depth + 1, valid⟩

private theorem materializes_array_iff {codecs : ScalarCodecs}
    {values : List Value} {results : List Wire} :
    Materializes codecs (.array values) (.array results) ↔
      All₂ (Materializes codecs) values results := by
  rw [materializes_succ]
  simp only [materializesAt]
  exact (all₂_uniform (P := fun depth a b => materializesAt depth codecs a b)
    (fun increase valid => materializesAt_mono increase valid)).symm

private theorem materializes_object_iff {codecs : ScalarCodecs}
    {entries : List (String × Value)} {results : List (String × Wire)} :
    Materializes codecs (.object [] entries) (.object results) ↔
      ∃ unsorted, All₂ (fun entry result => entry.1 = result.1 ∧
        Materializes codecs entry.2 result.2) entries unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ results = orderedMembers unsorted := by
  rw [materializes_succ]
  simp only [materializesAt]
  constructor
  · rintro ⟨depth, unsorted, related, unique, same⟩
    exact ⟨unsorted, all₂_mono (fun _ _ relation => ⟨relation.1, depth, relation.2⟩)
      related, unique, same⟩
  · rintro ⟨unsorted, related, unique, same⟩
    have existsEach : All₂ (fun (entry : String × Value) (result : String × Wire) =>
        ∃ depth, entry.1 = result.1 ∧ materializesAt depth codecs entry.2 result.2)
        entries unsorted := by
      apply all₂_mono (left := entries) (right := unsorted) _ related
      intro entry result relation
      obtain ⟨depth, materialized⟩ := relation.2
      exact ⟨depth, relation.1, materialized⟩
    obtain ⟨depth, bounded⟩ := (all₂_uniform (fun increase valid =>
      ⟨valid.1, materializesAt_mono increase valid.2⟩)).mp existsEach
    exact ⟨depth, unsorted, bounded, unique, same⟩

private theorem materializes_map_iff {codecs : ScalarCodecs}
    {entries : List (Scalar × Value)} {results : List (String × Wire)} :
    Materializes codecs (.map entries) (.object results) ↔
      ∃ unsorted, All₂ (fun entry result => KeySpelling codecs.numbers entry.1 result.1 ∧
        Materializes codecs entry.2 result.2) entries unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ results = orderedMembers unsorted := by
  rw [materializes_succ]
  simp only [materializesAt]
  constructor
  · rintro ⟨depth, unsorted, related, unique, same⟩
    exact ⟨unsorted, all₂_mono (fun _ _ relation => ⟨relation.1, depth, relation.2⟩)
      related, unique, same⟩
  · rintro ⟨unsorted, related, unique, same⟩
    have existsEach : All₂ (fun (entry : Scalar × Value) (result : String × Wire) =>
        ∃ depth, KeySpelling codecs.numbers entry.1 result.1 ∧
          materializesAt depth codecs entry.2 result.2) entries unsorted := by
      apply all₂_mono (left := entries) (right := unsorted) _ related
      intro entry result relation
      obtain ⟨depth, materialized⟩ := relation.2
      exact ⟨depth, relation.1, materialized⟩
    obtain ⟨depth, bounded⟩ := (all₂_uniform (fun increase valid =>
      ⟨valid.1, materializesAt_mono increase valid.2⟩)).mp existsEach
    exact ⟨depth, unsorted, bounded, unique, same⟩

private theorem all₂_congr_mem {R S : α → β → Prop} {left : List α} {right : List β}
    (same : ∀ a ∈ left, ∀ b, R a b ↔ S a b) : All₂ R left right ↔ All₂ S left right := by
  induction left generalizing right with
  | nil => simp [All₂]
  | cons a tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons b rest =>
      simp only [all₂_cons_iff]
      exact and_congr (same a (by simp) b)
        (ih (fun a member b => same a (by simp [member]) b))

private theorem all₂_map_left {R : β → γ → Prop} (f : α → β)
    (left : List α) (right : List γ) :
    All₂ R (left.map f) right ↔ All₂ (fun a b => R (f a) b) left right := by
  induction left generalizing right with
  | nil => simp [All₂]
  | cons a tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons b rest => simp only [List.map_cons, all₂_cons_iff, ih]

private theorem all₂_names {left : List (α × β)} {right : List (α × γ)}
    {R : β → γ → Prop}
    (related : All₂ (fun a b => a.1 = b.1 ∧ R a.2 b.2) left right) :
    left.map Prod.fst = right.map Prod.fst := by
  induction left generalizing right with
  | nil => cases right <;> simp_all [All₂]
  | cons a tail ih =>
    cases right with
    | nil => simp [All₂] at related
    | cons b rest =>
      have both := all₂_cons_iff.mp related
      simp only [List.map_cons]
      rw [both.1.1, ih both.2]

private theorem all₂_pair {R : α → β → Prop} {S : α → γ → Prop}
    (left : List α) (right : List (β × γ)) :
    All₂ (fun a b => R a b.1 ∧ S a b.2) left right ↔
      All₂ R left (right.map Prod.fst) ∧ All₂ S left (right.map Prod.snd) := by
  induction left generalizing right with
  | nil => cases right <;> simp [All₂]
  | cons a tail ih =>
    cases right with
    | nil => simp [All₂]
    | cons b rest =>
      simp only [List.map_cons, all₂_cons_iff, ih]
      simp [and_assoc, and_left_comm]

private theorem all₂_zip {R : α → β → Prop} {S : α → γ → Prop}
    {left : List α} {names : List β} {values : List γ}
    (first : All₂ R left names) (second : All₂ S left values) :
    All₂ (fun a b => R a b.1 ∧ S a b.2) left (names.zip values) := by
  induction left generalizing names values with
  | nil => cases names <;> cases values <;> simp_all [All₂]
  | cons a tail ih =>
    cases names with
    | nil => simp [All₂] at first
    | cons b rest =>
      cases values with
      | nil => simp [All₂] at second
      | cons c more =>
        have f := all₂_cons_iff.mp first
        have s := all₂_cons_iff.mp second
        exact all₂_cons_iff.mpr ⟨⟨f.1, s.1⟩, ih f.2 s.2⟩

private theorem zip_fst_of_length {names : List α} {values : List β}
    (same : names.length = values.length) : (names.zip values).map Prod.fst = names := by
  induction names generalizing values with
  | nil => simp
  | cons a tail ih =>
    cases values with
    | nil => simp at same
    | cons b rest => simp_all

private theorem zip_projections (entries : List (α × β)) :
    (entries.map Prod.fst).zip (entries.map Prod.snd) = entries := by
  induction entries with
  | nil => rfl
  | cons entry rest ih => simp [ih]


private theorem materializes_object_full {codecs : ScalarCodecs}
    {entries : List (String × Value)} {wire : Wire} :
    Materializes codecs (.object [] entries) wire ↔
      ∃ unsorted, All₂ (fun entry result => entry.1 = result.1 ∧
        Materializes codecs entry.2 result.2) entries unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ wire = .object (orderedMembers unsorted) := by
  cases wire <;> try solve | simp [materializes_succ, materializesAt]
  case object results =>
    rw [materializes_object_iff]
    simp only [Wire.object.injEq]

private theorem materializes_map_full {codecs : ScalarCodecs}
    {entries : List (Scalar × Value)} {wire : Wire} :
    Materializes codecs (.map entries) wire ↔
      ∃ unsorted, All₂ (fun entry result => KeySpelling codecs.numbers entry.1 result.1 ∧
        Materializes codecs entry.2 result.2) entries unsorted ∧
        (unsorted.map Prod.fst).Nodup ∧ wire = .object (orderedMembers unsorted) := by
  cases wire <;> try solve | simp [materializes_succ, materializesAt]
  case object results =>
    rw [materializes_map_iff]
    simp only [Wire.object.injEq]

/-- The executable materializer implements the independent canonical relation
for every finite source tree. Codec semantics remain explicit parameters. -/
theorem materialize_iff_Materializes (codecs : ScalarCodecs) (value : Value) (wire : Wire) :
    materialize codecs value = .ok wire ↔ Materializes codecs value wire := by
  cases value with
  | null | nilBytes =>
    cases wire <;> simp [materialize, materializes_succ, materializesAt, eq_comm]
  | nilArray =>
    cases wire <;> simp [materialize, materializes_succ, materializesAt, eq_comm]
    case array items => cases items <;> simp
  | nilMap =>
    cases wire <;> simp [materialize, materializes_succ, materializesAt, eq_comm]
    case object entries => cases entries <;> simp
  | host identity payload =>
    simp only [materialize]
    rw [materialize_iff_Materializes codecs payload wire]
    rw [materializes_succ (value := .host identity payload)]
    simp only [materializesAt]
    rfl
  | scalar scalar =>
    rw [materializes_succ]
    simp only [materializesAt, exists_const]
    by_cases valid : jsonValid codecs.numbers (encodeScalar codecs .json scalar) = true
    · simp only [materialize, valid, ite_true, Except.ok.injEq]
      constructor
      · intro same
        subst wire
        exact ⟨canonicalScalar_exists codecs .json scalar, (jsonValid_iff_JSONValid ..).mp valid⟩
      · intro relation
        exact canonicalScalar_realized relation.1
    · simp only [materialize, valid, Bool.false_eq_true, ite_false]
      simp only [reduceCtorEq, false_iff]
      rintro ⟨canonical, json⟩
      have same := canonicalScalar_realized canonical
      subst wire
      exact valid ((jsonValid_iff_JSONValid ..).mpr json)
  | array values =>
    have mapped (results : List Wire) :
        values.mapM (materialize codecs) = .ok results ↔ All₂ (Materializes codecs) values results :=
      (mapM_ok_iff ..).trans (all₂_congr_mem (fun child member result =>
        materialize_iff_Materializes codecs child result))
    cases wire <;>
      simp only [materialize, attach_mapM_eq, except_bind_ok, pure, Except.pure]
    all_goals try solve | simp [materializes_succ, materializesAt]
    case array =>
      simp only [Except.ok.injEq, Wire.array.injEq, exists_eq_right]
      exact (mapped _).trans materializes_array_iff.symm
  | object fields extra =>
    cases fields with
    | cons field rest => cases wire <;> simp [materialize, materializes_succ, materializesAt]
    | nil =>
      have mapped (results : List (String × Wire)) :
          extra.mapM (fun entry => do
            let child ← materialize codecs entry.2
            pure (entry.1, child)) = .ok results ↔
          All₂ (fun entry result => entry.1 = result.1 ∧
            Materializes codecs entry.2 result.2) extra results := by
        rw [mapM_ok_iff]
        apply all₂_congr_mem
        intro entry member result
        rcases result with ⟨name, result⟩
        simp only [except_bind_ok, pure, Except.pure, Except.ok.injEq, Prod.mk.injEq]
        constructor
        · rintro ⟨child, done, same, childSame⟩
          subst child
          exact ⟨same, (materialize_iff_Materializes codecs entry.2 result).mp done⟩
        · rintro ⟨same, relation⟩
          exact ⟨result, (materialize_iff_Materializes codecs entry.2 result).mpr relation,
            same, rfl⟩
      rw [materializes_object_full]
      simp only [materialize]
      rw [attach_mapM_eq (fun entry : String × Value => do
        let child ← materialize codecs entry.2
        pure (entry.1, child)) extra]
      by_cases unique : (extra.map Prod.fst).Nodup
      · simp only [unique, decide_true, Bool.not_true, Bool.false_eq_true, ite_false,
          except_bind_ok, pure, Except.pure, Except.ok.injEq]
        constructor
        · rintro ⟨results, done, same⟩
          have related := (mapped results).mp done
          exact ⟨results, related, (all₂_names related) ▸ unique, same.symm⟩
        · rintro ⟨results, related, _, same⟩
          exact ⟨results, (mapped results).mpr related, same.symm⟩
      · simp only [unique, decide_false, Bool.not_false, ite_true, reduceCtorEq, false_iff]
        rintro ⟨results, related, distinct, _⟩
        exact unique ((all₂_names related).symm ▸ distinct)
  | map entries =>
    have mapped (results : List Wire) :
        entries.mapM (fun entry => materialize codecs entry.2) = .ok results ↔
        All₂ (fun entry result => Materializes codecs entry.2 result) entries results :=
      (mapM_ok_iff ..).trans (all₂_congr_mem (fun entry member result =>
        materialize_iff_Materializes codecs entry.2 result))
    rw [materializes_map_full]
    simp only [materialize]
    rw [attach_mapM_eq (fun entry : Scalar × Value => materialize codecs entry.2) entries]
    simp only [except_bind_ok, pure, Except.pure, Except.ok.injEq]
    constructor
    · rintro ⟨names, named, values, done, same⟩
      have keys := (nameKeys_iff ..).mp named
      have keys' := (all₂_map_left Prod.fst entries names).mp keys.2
      have values' := (mapped values).mp done
      have lengths : names.length = values.length := keys'.1.symm.trans values'.1
      exact ⟨names.zip values, all₂_zip keys' values',
        (zip_fst_of_length lengths).symm ▸ keys.1, same.symm⟩
    · rintro ⟨results, related, unique, same⟩
      have pairs := (all₂_pair entries results).mp related
      refine ⟨results.map Prod.fst, (nameKeys_iff ..).mpr ⟨unique, ?_⟩,
        results.map Prod.snd, (mapped _).mpr pairs.2, ?_⟩
      · exact (all₂_map_left Prod.fst entries _).mpr pairs.1
      · rw [zip_projections]; exact same.symm
  | absent | jsonSnapshot | any | union =>
    cases wire <;> simp [materialize, materializes_succ, materializesAt]
termination_by sizeOf value
decreasing_by
  all_goals first
  | (simp only [Value.host.sizeOf_spec]; omega)
  | (
    have smaller := List.sizeOf_lt_of_mem member
    simp only [Value.array.sizeOf_spec, Value.object.sizeOf_spec, Value.map.sizeOf_spec, List.nil.sizeOf_spec]
    try cases entry
    try simp only [Prod.mk.sizeOf_spec] at smaller ⊢
    omega)

end ValueContract.Candidate
