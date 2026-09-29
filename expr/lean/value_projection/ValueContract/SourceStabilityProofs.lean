import ValueContract.SourceGraphBudgetProofs
import ValueContract.SourceRawOutcomeProofs
import ValueContract.SourcePreferenceProofs

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

private theorem bind_congr_ok {first : Except Failure α} {left right : α → Except Failure β}
    (same : ∀ value, first = .ok value → left value = right value) :
    (first >>= left) = (first >>= right) := by
  cases first with
  | error failure => rfl
  | ok value => exact same value rfl

private theorem selectedMemberInput_mem {member entries input}
    (selected : selectedMemberInput member entries = some input) :
    ∃ entry ∈ entries, entry.2 = input := by
  unfold selectedMemberInput at selected
  split at selected
  · rename_i entry found
    cases selected
    exact ⟨entry, List.mem_of_find?_eq_some found, rfl⟩
  · cases found : entries.find? (fun entry =>
        entry.1 == member.sourceName && hasSuppliedValue entry.2) with
    | none => simp [found] at selected
    | some entry =>
      simp only [found, Option.map_some, Option.some.injEq] at selected
      exact ⟨entry, List.mem_of_find?_eq_some found, selected⟩

private theorem objectMemberResult_congr {complete left right entries member}
    (same : ∀ entry ∈ entries, left member.child entry.2 = right member.child entry.2) :
    objectMemberResult complete left entries member = objectMemberResult complete right entries member := by
  unfold objectMemberResult
  have aliases : (entries.filter (fun entry => hasSuppliedValue entry.2 &&
      (entry.1 == member.sourceName || entry.1 == member.wireAlias))).map
      (fun entry => left member.child entry.2) =
      (entries.filter (fun entry => hasSuppliedValue entry.2 &&
      (entry.1 == member.sourceName || entry.1 == member.wireAlias))).map
      (fun entry => right member.child entry.2) := by
    apply List.map_congr_left
    intro entry inside
    exact same entry (List.mem_filter.mp inside).1
  dsimp only
  rw [aliases]
  apply bind_congr_ok
  intro ignored done
  split
  · rfl
  · rename_i input selected
    obtain ⟨entry, inside, child⟩ := selectedMemberInput_mem selected
    have equal := same entry inside
    rw [child] at equal
    rw [equal]

private def normalizeKey (checks : ExternalScalarChecks) (kind : MapKeyKind)
    (rules : ScalarRules) (entry : Scalar × Input) : Except Failure (Scalar × Input) := do
  let key ← match kind with
    | .builtin => .ok entry.1
    | .scalar expected => match coerceScalar expected entry.1 with
      | some key => .ok key
      | none => .error .invalid
  if !mapKeyCompatible kind key || !scalarAllowed checks rules key then throw .invalid
  return (key, entry.2)

private theorem normalizeKey_child {checks kind rules entry normalized}
    (done : normalizeKey checks kind rules entry = .ok normalized) : entry.2 = normalized.2 := by
  cases kind <;> simp only [normalizeKey] at done
  all_goals simp only [Bind.bind, Except.bind] at done
  all_goals repeat' first
    | split at done
    | (simp_all [throw]; done)
  all_goals cases done; rfl

private theorem normalized_child_depth {checks kind rules input entries normalized entry}
    (extracted : mapEntries input = some entries)
    (done : combineChecked (entries.map (normalizeKey checks kind rules)) = .ok normalized)
    (inside : entry ∈ normalized) : inputDepth entry.2 < inputDepth input := by
  obtain ⟨original, member, related⟩ := all₂_right ((combineChecked_map_iff ..).mp done) inside
  rw [← normalizeKey_child related]
  exact mapEntries_child_depth extracted member

/-- Source-body execution depends on adequate child results and preferences,
not on the numeric counters used to obtain them. Rejected children are retained
in this equality just as successful children are. -/
theorem resolveBody_stable {declarations checks keys firstDepth secondDepth firstRank secondRank
    complete contract input firstSame secondSame firstDescend secondDescend}
    (firstEnough : inputDepth input ≤ firstDepth + 1)
    (secondEnough : inputDepth input ≤ secondDepth + 1)
    (same : ∀ child ∈ nonconsumingChildren contract, ∀ mode,
      firstSame mode child input = secondSame mode child input)
    (descend : ∀ child, child ∈ consumingChildren contract ∨ child ∈ nonconsumingChildren contract →
      ∀ mode next, inputDepth next < inputDepth input →
        firstDescend mode child next = secondDescend mode child next)
    (preference : ∀ child ∈ nonconsumingChildren contract,
      prefersObjectInput declarations (firstRank + 1) child input =
        prefersObjectInput declarations (secondRank + 1) child input) :
    resolveBody declarations checks keys firstDepth firstRank complete contract input firstSame firstDescend =
      resolveBody declarations checks keys secondDepth secondRank complete contract input secondSame secondDescend := by
  unfold resolveBody
  split
  · rfl
  · rfl
  · cases contract <;> simp only
    case any => rw [resolveRawAt_stable firstEnough secondEnough]
    case alias child => exact same child (by simp [nonconsumingChildren]) complete
    case nullable child =>
      split
      · rfl
      · exact same child (by simp [nonconsumingChildren]) complete
    case nonNull child =>
      split
      · rfl
      · exact same child (by simp [nonconsumingChildren]) complete
    case array child bounds =>
      split
      · rfl
      · rename_i items shape
        have children : items.map (firstDescend complete child) = items.map (secondDescend complete child) := by
          apply List.map_congr_left
          intro item inside
          exact descend child (by simp [consumingChildren]) complete item (stripped_array_child_depth shape inside)
        rw [children]
      · rfl
    case map kind rules child bounds =>
      split
      · rfl
      · cases extracted : mapEntries input with
        | none => rfl
        | some entries =>
          simp only [Bind.bind, Except.bind]
          split
          · rfl
          · change ((do
              let normalized ← combineChecked (entries.map (normalizeKey checks kind rules))
              let _ ← nameKeys keys (normalized.map Prod.fst)
              if !(decide (normalized.Pairwise (fun left right => scalarEqual left.1 right.1 = false))) then
                throw .invalid
              let values ← combineChecked (normalized.map fun entry => do
                let value ← firstDescend complete child entry.2
                pure (entry.1, value))
              pure (Resolution.mk (.map (values.map (fun entry => (entry.1, entry.2.value))))
                (values.flatMap (fun entry => entry.2.missing)))) : ResolveResult) = (do
              let normalized ← combineChecked (entries.map (normalizeKey checks kind rules))
              let _ ← nameKeys keys (normalized.map Prod.fst)
              if !(decide (normalized.Pairwise (fun left right => scalarEqual left.1 right.1 = false))) then
                throw .invalid
              let values ← combineChecked (normalized.map fun entry => do
                let value ← secondDescend complete child entry.2
                pure (entry.1, value))
              pure (Resolution.mk (.map (values.map (fun entry => (entry.1, entry.2.value))))
                (values.flatMap (fun entry => entry.2.missing))))
            apply bind_congr_ok
            intro normalized done
            have children : normalized.map (fun entry => do
                let value ← firstDescend complete child entry.2
                pure (entry.1, value)) = normalized.map (fun entry => do
                let value ← secondDescend complete child entry.2
                pure (entry.1, value)) := by
              apply List.map_congr_left
              intro entry inside
              rw [descend child (by simp [consumingChildren]) complete entry.2
                (normalized_child_depth extracted done inside)]
            rw [children]
    case object members isOpen =>
      cases extracted : objectEntries input with
      | none => rfl
      | some entries =>
        simp only [Bind.bind, Except.bind]
        have fields : members.map (objectMemberResult complete (firstDescend complete) entries) =
            members.map (objectMemberResult complete (secondDescend complete) entries) := by
          apply List.map_congr_left
          intro member inside
          apply objectMemberResult_congr
          intro entry supplied
          exact descend member.child (Or.inl (List.mem_map_of_mem inside)) complete entry.2
            (objectEntries_child_depth extracted supplied)
        have extras : (entries.filter (fun entry => !members.any (fun member =>
            entry.1 == member.sourceName || entry.1 == member.wireAlias))).map (fun entry => do
              let value ← resolveRawAt keys firstDepth entry.2
              pure (entry.1, value)) =
            (entries.filter (fun entry => !members.any (fun member =>
            entry.1 == member.sourceName || entry.1 == member.wireAlias))).map (fun entry => do
              let value ← resolveRawAt keys secondDepth entry.2
              pure (entry.1, value)) := by
          apply List.map_congr_left
          intro entry inside
          have smaller := objectEntries_child_depth extracted (List.mem_filter.mp inside).1
          rw [resolveRawAt_stable (first := firstDepth) (second := secondDepth)
            (input := entry.2) (by omega) (by omega)]
        simp only [Bind.bind, Except.bind] at extras
        rw [fields, extras]
    case union occurrence alternatives =>
      split
      · rename_i actual branch payload shape
        split
        · rfl
        · split
          · rfl
          · rename_i alternative selected
            unfold wrapBranch
            rw [descend alternative.child
              (Or.inr (List.mem_map_of_mem (List.mem_of_find?_eq_some selected))) complete payload
              (stripped_selected_child_depth shape)]
      · have candidates (mode : Bool) : alternatives.map (fun alternative => {
              branch := alternative.identity,
              preferred := prefersObjectInput declarations (firstRank + 1) alternative.child input,
              result := firstSame mode alternative.child input : BranchCandidate }) =
            alternatives.map (fun alternative => {
              branch := alternative.identity,
              preferred := prefersObjectInput declarations (secondRank + 1) alternative.child input,
              result := secondSame mode alternative.child input : BranchCandidate }) := by
          apply List.map_congr_left
          intro alternative inside
          have edge : alternative.child ∈ nonconsumingChildren (.union occurrence alternatives) :=
            List.mem_map_of_mem inside
          rw [preference alternative.child edge, same alternative.child edge mode]
        rw [candidates true, candidates false]

/-- Every full resolution outcome is stable above the independently derived
input/rank bounds. This includes ambiguity, invalid input and unsupported data;
no branch can become eligible merely by increasing a traversal cutoff. -/
theorem resolveAt_stable {declarations checks keys firstDepth secondDepth firstRank secondRank
    complete identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (firstDepthEnough : inputDepth input ≤ firstDepth)
    (secondDepthEnough : inputDepth input ≤ secondDepth)
    (firstRankEnough : declaration.expansionRank < firstRank)
    (secondRankEnough : declaration.expansionRank < secondRank) :
    resolveAt declarations checks keys firstDepth firstRank complete identity input =
      resolveAt declarations checks keys secondDepth secondRank complete identity input := by
  induction firstDepth generalizing secondDepth firstRank secondRank complete identity input declaration with
  | zero => have positive := inputDepth_pos input; omega
  | succ firstDepth depthIH =>
    cases secondDepth with
    | zero => have positive := inputDepth_pos input; omega
    | succ secondDepth =>
      induction firstRank generalizing secondRank complete identity input declaration with
      | zero => omega
      | succ firstRank rankIH =>
        cases secondRank with
        | zero => omega
        | succ secondRank =>
          have body := resolveBody_stable (declarations := declarations) (checks := checks) (keys := keys)
            (firstRank := firstRank) (secondRank := secondRank) (complete := complete)
            (contract := declaration.contract) (input := input)
            (firstSame := fun mode child value => resolveAt declarations checks keys (firstDepth + 1) firstRank mode child value)
            (secondSame := fun mode child value => resolveAt declarations checks keys (secondDepth + 1) secondRank mode child value)
            (firstDescend := fun mode child value => resolveAt declarations checks keys firstDepth
              (maximumExpansionRank declarations + 1) mode child value)
            (secondDescend := fun mode child value => resolveAt declarations checks keys secondDepth
              (maximumExpansionRank declarations + 1) mode child value)
            firstDepthEnough secondDepthEnough
          have same : ∀ child ∈ nonconsumingChildren declaration.contract, ∀ mode,
              resolveAt declarations checks keys (firstDepth + 1) firstRank mode child input =
                resolveAt declarations checks keys (secondDepth + 1) secondRank mode child input := by
            intro child edge mode
            obtain ⟨childDeclaration, childFound, smaller⟩ := declaration_nonconsuming_rank wellFormed found edge
            exact rankIH childFound firstDepthEnough (by omega) (by omega) secondDepthEnough
          have descend : ∀ child, child ∈ consumingChildren declaration.contract ∨
              child ∈ nonconsumingChildren declaration.contract → ∀ mode next,
              inputDepth next < inputDepth input →
              resolveAt declarations checks keys firstDepth (maximumExpansionRank declarations + 1) mode child next =
                resolveAt declarations checks keys secondDepth (maximumExpansionRank declarations + 1) mode child next := by
            intro child edge mode next smaller
            have existsChild : ∃ childDeclaration, findDeclaration declarations child = some childDeclaration := by
              rcases edge with consumed | expanded
              · exact declaration_consuming_exists wellFormed found consumed
              · obtain ⟨childDeclaration, childFound, _⟩ := declaration_nonconsuming_rank wellFormed found expanded
                exact ⟨childDeclaration, childFound⟩
            obtain ⟨childDeclaration, childFound⟩ := existsChild
            have bound := declaration_rank_le_maximum (findDeclaration_mem childFound)
            exact depthIH childFound (by omega) (by omega) (by omega) (by omega)
          have preference : ∀ child ∈ nonconsumingChildren declaration.contract,
              prefersObjectInput declarations (firstRank + 1) child input =
                prefersObjectInput declarations (secondRank + 1) child input := by
            intro child edge
            obtain ⟨childDeclaration, childFound, smaller⟩ := declaration_nonconsuming_rank wellFormed found edge
            apply Bool.eq_iff_iff.mpr
            exact (prefersObjectInput_iff wellFormed childFound (by omega)).trans
              (prefersObjectInput_iff wellFormed childFound (by omega)).symm
          simp only [resolveAt, found, Bind.bind, Except.bind]
          rw [body same descend preference]

/-- Public resolution is the same as any adequately budgeted worker run, so
semantic outcomes can quantify adequate finite derivations independently. -/
theorem resolve_eq_resolveAt {declarations checks keys role identity input declaration depth rank}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (depthEnough : inputDepth input ≤ depth)
    (rankEnough : declaration.expansionRank < rank) :
    resolve declarations checks keys role identity input =
      resolveAt declarations checks keys depth rank
        (role == .enumMember || role == .defaultValue) identity input := by
  have valid := (validateDeclarations_iff declarations).mpr wellFormed
  simp only [resolve, valid, Bool.not_true, Bool.false_eq_true, ↓reduceIte]
  apply resolveAt_stable wellFormed found (by omega) depthEnough _ rankEnough
  have bound := declaration_rank_le_maximum (findDeclaration_mem found)
  omega

end ValueContract.Candidate
