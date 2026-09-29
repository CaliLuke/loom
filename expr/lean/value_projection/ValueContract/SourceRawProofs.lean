import ValueContract.ResolverProofs

namespace ValueContract.Candidate

theorem inputDepth_pos (input : Input) : 0 < inputDepth input := by
  cases input <;> simp [inputDepth] <;> omega

theorem array_inputDepth_lt {items : List Input} {child : Input}
    (member : child ∈ items) : inputDepth child < inputDepth (.array items) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := inputDepth) member) 0
  simp only [inputDepth]
  omega

theorem object_inputDepth_lt {entries : List (String × Input)} {entry : String × Input}
    (member : entry ∈ entries) : inputDepth entry.2 < inputDepth (.object entries) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => inputDepth entry.2) member) 0
  simp only [inputDepth]
  omega

theorem map_inputDepth_lt {entries : List (Scalar × Input)} {entry : Scalar × Input}
    (member : entry ∈ entries) : inputDepth entry.2 < inputDepth (.map entries) := by
  have bound := foldMax_member (List.mem_map_of_mem (f := fun entry => inputDepth entry.2) member) 0
  simp only [inputDepth]
  omega

/-- Raw source interpretation is stable under more derivation depth. This
quantifies over arbitrary finite objects, maps and arrays, including host evidence. -/
theorem RawResolutionAt_mono {keys : KeyCodec} {low high : Nat} {input : Input} {value : Value}
    (increase : low ≤ high) (related : RawResolutionAt keys low input value) :
    RawResolutionAt keys high input value := by
  induction low generalizing high input value with
  | zero => simp [RawResolutionAt] at related
  | succ low ih =>
    cases high with
    | zero => omega
    | succ high =>
      have smaller : low ≤ high := by omega
      cases input <;> cases value <;> simp only [RawResolutionAt] at related ⊢
      all_goals try exact related
      case array.array inputs values =>
        exact ⟨related.1, fun pair member => ih smaller (related.2 pair member)⟩
      case object.object inputs fields values =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro pair member
        have child := related.2.2.2 pair member
        exact ⟨child.1, ih smaller child.2⟩
      case map.map inputs values =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro pair member
        have child := related.2.2.2 pair member
        exact ⟨child.1, ih smaller child.2⟩
      case host.host identity payload other child => exact ⟨related.1, ih smaller related.2⟩

/-- An independent raw derivation needs at most the actual input's structural
height. The theorem does not assume the resolver has already succeeded. -/
theorem RawResolutionAt_rebudget {keys : KeyCodec} {depth budget : Nat}
    {input : Input} {value : Value} (related : RawResolutionAt keys depth input value)
    (enough : inputDepth input ≤ budget) : RawResolutionAt keys budget input value := by
  induction depth generalizing budget input value with
  | zero => simp [RawResolutionAt] at related
  | succ depth ih =>
    cases budget with
    | zero => have positive := inputDepth_pos input; omega
    | succ budget =>
      cases input <;> cases value <;> simp only [RawResolutionAt] at related ⊢
      all_goals try exact related
      case array.array inputs values =>
        refine ⟨related.1, ?_⟩
        intro pair member
        apply ih (related.2 pair member)
        have small := array_inputDepth_lt (List.of_mem_zip member).1
        omega
      case object.object inputs fields values =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro pair member
        have child := related.2.2.2 pair member
        refine ⟨child.1, ih child.2 ?_⟩
        have small := object_inputDepth_lt (List.of_mem_zip member).1
        omega
      case map.map inputs values =>
        refine ⟨related.1, related.2.1, related.2.2.1, ?_⟩
        intro pair member
        have child := related.2.2.2 pair member
        refine ⟨child.1, ih child.2 ?_⟩
        have small := map_inputDepth_lt (List.of_mem_zip member).1
        omega
      case host.host identity payload other child =>
        refine ⟨related.1, ih related.2 ?_⟩
        simp only [inputDepth] at enough
        omega

/-- At the computed input-height budget the executable raw interpreter is
equivalent to the unbounded independent judgment, in both directions. -/
theorem resolveRaw_iff (keys : KeyCodec) (input : Input) (value : Value) :
    resolveRawAt keys (inputDepth input) input = .ok value ↔ RawResolution keys input value := by
  rw [resolveRawAt_iff]
  exact ⟨fun related => ⟨_, related⟩,
    fun ⟨_, related⟩ => RawResolutionAt_rebudget related (Nat.le_refl _)⟩

/-- Raw built-in snapshots have no absent slot, executable cycle, opaque value,
or trusted union-selection marker at their outermost host-transparent shape. -/
def RawInputSupported (input : Input) : Prop :=
  match stripHostInput input with
  | .absent | .cycle _ | .opaque _ | .selected _ _ _ => False
  | _ => True

theorem rawResolution_inputSupported {keys : KeyCodec} {depth : Nat} {input : Input} {value : Value}
    (related : RawResolutionAt keys depth input value) : RawInputSupported input := by
  induction depth generalizing input value with
  | zero => simp [RawResolutionAt] at related
  | succ depth ih =>
    cases input <;> cases value <;> simp [RawResolutionAt] at related <;>
      simp [RawInputSupported, stripHostInput]
    case host.host => simpa only [RawInputSupported, stripHostInput] using ih related.2

/-- Inspecting outer host evidence cannot increase the authored input height;
all nested child evidence remains inside that finite measure. -/
theorem stripHostInput_depth_le (input : Input) :
    inputDepth (stripHostInput input) ≤ inputDepth input := by
  induction input using stripHostInput.induct with
  | case1 identity payload ih =>
    simp only [stripHostInput, inputDepth]
    omega
  | case2 input notHost => simp [stripHostInput]

theorem stripped_array_child_depth {input : Input} {items : List Input} {child : Input}
    (shape : stripHostInput input = .array items) (inside : child ∈ items) :
    inputDepth child < inputDepth input := by
  have outer := stripHostInput_depth_le input
  rw [shape] at outer
  exact Nat.lt_of_lt_of_le (array_inputDepth_lt inside) outer

theorem stripped_object_child_depth {input : Input} {entries : List (String × Input)}
    {entry : String × Input} (shape : stripHostInput input = .object entries)
    (inside : entry ∈ entries) : inputDepth entry.2 < inputDepth input := by
  have outer := stripHostInput_depth_le input
  rw [shape] at outer
  exact Nat.lt_of_lt_of_le (object_inputDepth_lt inside) outer

theorem stripped_map_child_depth {input : Input} {entries : List (Scalar × Input)}
    {entry : Scalar × Input} (shape : stripHostInput input = .map entries)
    (inside : entry ∈ entries) : inputDepth entry.2 < inputDepth input := by
  have outer := stripHostInput_depth_le input
  rw [shape] at outer
  exact Nat.lt_of_lt_of_le (map_inputDepth_lt inside) outer

theorem stripped_selected_child_depth {input payload : Input} {occurrence branch : Identity}
    (shape : stripHostInput input = .selected occurrence branch payload) :
    inputDepth payload < inputDepth input := by
  have outer := stripHostInput_depth_le input
  rw [shape] at outer
  simp only [inputDepth] at outer
  omega

end ValueContract.Candidate
