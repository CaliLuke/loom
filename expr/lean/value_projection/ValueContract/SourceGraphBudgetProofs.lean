import ValueContract.SourceBudgetProofs
import ValueContract.SourceBodyProofs

set_option maxHeartbeats 2000000

namespace ValueContract.Candidate

private theorem exceptThrow (failure : Failure) :
    (throw failure : Except Failure α) = .error failure := rfl

private theorem bind_safe {first : Except Failure α} {next : α → Except Failure β}
    (safe : NoMalformed first)
    (steps : ∀ value, first = .ok value → NoMalformed (next value)) :
    NoMalformed (first >>= next) := by
  cases first with
  | error failure => simpa [NoMalformed, Bind.bind, Except.bind] using safe
  | ok value => exact steps value rfl

private theorem combined_map_safe (f : α → Except Failure β) (items : List α)
    (safe : ∀ item ∈ items, NoMalformed (f item)) :
    NoMalformed (combineChecked (items.map f)) := by
  apply combineChecked_noMalformed
  intro result inside
  obtain ⟨item, member, rfl⟩ := List.mem_map.mp inside
  exact safe item member

/-- Map extraction retains a strict input child, including object spelling and
transparent host evidence; normalized keys cannot enlarge the recursion budget. -/
theorem mapEntries_child_depth {input entries entry}
    (extracted : mapEntries input = some entries) (inside : entry ∈ entries) :
    inputDepth entry.2 < inputDepth input := by
  have shape := (mapEntries_iff ..).mp extracted
  unfold MapInputEntries at shape
  split at shape <;> try contradiction
  · subst entries
    exact stripped_map_child_depth (by assumption) inside
  · subst entries
    obtain ⟨field, member, rfl⟩ := List.mem_map.mp inside
    exact stripped_object_child_depth (by assumption) member

/-- Object extraction retains each exact supplied input child, even when an
all-string-key map is interpreted as an object. -/
theorem objectEntries_child_depth {input entries entry}
    (extracted : objectEntries input = some entries) (inside : entry ∈ entries) :
    inputDepth entry.2 < inputDepth input := by
  have shape := (objectEntries_iff ..).mp extracted
  unfold ObjectInputEntries at shape
  split at shape <;> try contradiction
  · subst entries
    exact stripped_object_child_depth (by assumption) inside
  · obtain ⟨original, member, related⟩ := all₂_right shape inside
    rw [← related.2]
    exact stripped_map_child_depth (by assumption) member

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

private theorem objectMemberResult_safe {complete resolveChild entries member}
    (safe : ∀ entry ∈ entries, NoMalformed (resolveChild member.child entry.2)) :
    NoMalformed (objectMemberResult complete resolveChild entries member) := by
  unfold objectMemberResult
  apply NoMalformed_bind
  · apply combined_map_safe
    intro entry inside
    exact safe entry (List.mem_filter.mp inside).1
  · intro ignored
    split
    · cases complete <;> cases member.required <;> simp [NoMalformed, exceptThrow, Bind.bind, Except.bind]
    · rename_i input selected
      obtain ⟨entry, inside, same⟩ := selectedMemberInput_mem selected
      apply NoMalformed_bind
      · simpa [same] using safe entry inside
      · intro resolution; simp [NoMalformed]

private theorem rankCandidates_safe (candidates : List BranchCandidate) :
    NoMalformed (rankCandidates candidates) := by
  unfold rankCandidates
  unfold NoMalformed
  split <;> try simp
  split <;> simp

private theorem rankedWrapped_safe (candidates : List BranchCandidate) (occurrence : Identity)
    (safe : ∀ candidate ∈ candidates, NoMalformed candidate.result) :
    NoMalformed (do let chosen ← rankCandidates candidates; wrapBranch occurrence chosen) := by
  apply bind_safe (rankCandidates_safe candidates)
  intro candidate selected
  unfold wrapBranch
  apply NoMalformed_bind (safe candidate (rankCandidates_chosen selected))
  intro result; simp [NoMalformed]

private theorem completeWrapped_safe (complete : List BranchCandidate)
    (fallback : Unit → List BranchCandidate) (occurrence : Identity)
    (completeSafe : ∀ candidate ∈ complete, NoMalformed candidate.result)
    (fallbackSafe : ∀ candidate ∈ fallback (), NoMalformed candidate.result) :
    NoMalformed (do let chosen ← rankCompleteFirst complete fallback; wrapBranch occurrence chosen) := by
  unfold rankCompleteFirst
  split
  · exact rankedWrapped_safe _ _ fallbackSafe
  · exact rankedWrapped_safe _ _ completeSafe

private def normalizeKey (checks : ExternalScalarChecks) (kind : MapKeyKind)
    (rules : ScalarRules) (entry : Scalar × Input) : Except Failure (Scalar × Input) := do
  let key ← match kind with
    | .builtin => .ok entry.1
    | .scalar expected => match coerceScalar expected entry.1 with
      | some key => .ok key
      | none => .error .invalid
  if !mapKeyCompatible kind key || !scalarAllowed checks rules key then throw .invalid
  return (key, entry.2)

private theorem normalizeKey_safe (checks kind rules entry) :
    NoMalformed (normalizeKey checks kind rules entry) := by
  cases kind <;> simp only [normalizeKey]
  all_goals simp only [Bind.bind, Except.bind, exceptThrow]
  all_goals repeat' first
    | split
    | (simp [NoMalformed]; done)

private theorem normalizeKey_child {checks kind rules entry normalized}
    (done : normalizeKey checks kind rules entry = .ok normalized) : entry.2 = normalized.2 := by
  cases kind <;> simp only [normalizeKey] at done
  all_goals simp only [Bind.bind, Except.bind, exceptThrow] at done
  all_goals repeat' first
    | split at done
    | (simp_all; done)
  all_goals cases done; rfl

private theorem normalized_child_depth {checks kind rules input entries normalized entry}
    (extracted : mapEntries input = some entries)
    (done : combineChecked (entries.map (normalizeKey checks kind rules)) = .ok normalized)
    (inside : entry ∈ normalized) : inputDepth entry.2 < inputDepth input := by
  obtain ⟨original, member, related⟩ := all₂_right ((combineChecked_map_iff ..).mp done) inside
  rw [← normalizeKey_child related]
  exact mapEntries_child_depth extracted member

/-- Each source body preserves the graph/input recursion discipline, including
failed aliases, additional fields, and candidates not chosen by union ranking. -/
theorem resolveBody_noMalformed {declarations checks keys depth rank complete contract input same descend}
    (enough : inputDepth input ≤ depth + 1)
    (sameSafe : ∀ child ∈ nonconsumingChildren contract, ∀ mode, NoMalformed (same mode child input))
    (descendSafe : ∀ child, child ∈ consumingChildren contract ∨ child ∈ nonconsumingChildren contract →
      ∀ mode next, inputDepth next < inputDepth input → NoMalformed (descend mode child next)) :
    NoMalformed (resolveBody declarations checks keys depth rank complete contract input same descend) := by
  unfold resolveBody
  split
  · simp [NoMalformed]
  · simp [NoMalformed]
  · cases contract <;> simp only
    case custom => simp [NoMalformed]
    case any =>
      apply NoMalformed_bind (resolveRaw_noMalformed enough)
      intro value; simp [NoMalformed]
    case alias child => exact sameSafe child (by simp [nonconsumingChildren]) complete
    case nullable child =>
      split
      · simp [NoMalformed]
      · exact sameSafe child (by simp [nonconsumingChildren]) complete
    case nonNull child =>
      split
      · simp [NoMalformed]
      · exact sameSafe child (by simp [nonconsumingChildren]) complete
    case scalar kind rules =>
      repeat' first
        | split
        | (simp [NoMalformed]; done)
    case array child bounds =>
      split
      · split <;> simp [NoMalformed]
      · rename_i items shape
        split
        · simp [NoMalformed, exceptThrow, Bind.bind, Except.bind]
        · apply NoMalformed_bind
          · apply combined_map_safe
            intro item inside
            apply descendSafe child (by simp [consumingChildren]) complete
            exact stripped_array_child_depth shape inside
          · intro values; simp [NoMalformed]
      · simp [NoMalformed]
    case map kind rules child bounds =>
      split
      · split <;> simp [NoMalformed]
      · cases extracted : mapEntries input with
        | none => simp [NoMalformed]
        | some entries =>
          simp only [Bind.bind, Except.bind]
          split
          · simp [NoMalformed, exceptThrow]
          · change NoMalformed (do
              let normalized ← combineChecked (entries.map (normalizeKey checks kind rules))
              let _ ← nameKeys keys (normalized.map Prod.fst)
              if !(decide (normalized.Pairwise (fun left right => scalarEqual left.1 right.1 = false))) then
                throw .invalid
              let values ← combineChecked (normalized.map fun entry => do
                let value ← descend complete child entry.2
                pure (entry.1, value))
              pure (Resolution.mk (.map (values.map (fun entry => (entry.1, entry.2.value))))
                (values.flatMap (fun entry => entry.2.missing))))
            apply bind_safe (combined_map_safe _ _ (fun entry _ => normalizeKey_safe checks kind rules entry))
            intro normalized done
            apply NoMalformed_bind (nameKeys_noMalformed keys _)
            intro ignored
            split
            · simp [NoMalformed, exceptThrow, Bind.bind, Except.bind]
            · apply NoMalformed_bind
              · apply combined_map_safe
                intro entry inside
                apply NoMalformed_bind
                · apply descendSafe child (by simp [consumingChildren]) complete
                  exact normalized_child_depth extracted done inside
                · intro resolution; simp [NoMalformed]
              · intro values; simp [NoMalformed]
    case object members isOpen =>
      cases extracted : objectEntries input with
      | none => simp [NoMalformed]
      | some entries =>
        simp only [Bind.bind, Except.bind]
        split
        · simp [NoMalformed, exceptThrow]
        · split
          · simp [NoMalformed, exceptThrow]
          · have fieldSafe : ∀ member ∈ members,
                NoMalformed (objectMemberResult complete (descend complete) entries member) := by
              intro member inside
              apply objectMemberResult_safe
              intro entry supplied
              apply descendSafe member.child
                (Or.inl (List.mem_map_of_mem inside)) complete
              exact objectEntries_child_depth extracted supplied
            have extraSafe : ∀ entry ∈ entries, NoMalformed (do
                let value ← resolveRawAt keys depth entry.2
                pure (entry.1, value)) := by
              intro entry inside
              apply NoMalformed_bind
              · apply resolveRaw_noMalformed
                have smaller := objectEntries_child_depth extracted inside
                omega
              · intro value; simp [NoMalformed]
            apply NoMalformed_bind
            · apply combineChecked_noMalformed
              intro outcome inside
              rcases List.mem_append.mp inside with field | extra
              · obtain ⟨result, resultInside, rfl⟩ := List.mem_map.mp field
                obtain ⟨member, memberInside, rfl⟩ := List.mem_map.mp resultInside
                exact NoMalformed_map _ (fieldSafe member memberInside)
              · obtain ⟨result, resultInside, rfl⟩ := List.mem_map.mp extra
                obtain ⟨entry, entryInside, rfl⟩ := List.mem_map.mp resultInside
                exact NoMalformed_map _ (extraSafe entry (List.mem_filter.mp entryInside).1)
            · intro ignored
              apply NoMalformed_bind (combined_map_safe _ _ fieldSafe)
              intro fields
              apply NoMalformed_bind
              · apply combined_map_safe
                intro entry inside
                exact extraSafe entry (List.mem_filter.mp inside).1
              · intro extras; simp [NoMalformed]
    case union occurrence alternatives =>
      split
      · rename_i actual branch payload shape
        split
        · simp [NoMalformed]
        · split
          · simp [NoMalformed]
          · rename_i alternative selected
            unfold wrapBranch
            apply NoMalformed_bind
            · apply descendSafe alternative.child
                (Or.inr (List.mem_map_of_mem (List.mem_of_find?_eq_some selected))) complete
              exact stripped_selected_child_depth shape
            · intro resolution; simp [NoMalformed]
      · apply completeWrapped_safe
        · intro candidate inside
          obtain ⟨alternative, declared, sameCandidate⟩ := List.mem_map.mp (List.mem_filter.mp inside).1
          subst candidate
          exact sameSafe alternative.child (List.mem_map_of_mem declared) true
        · intro candidate inside
          split at inside
          · simp at inside
          · obtain ⟨alternative, declared, sameCandidate⟩ := List.mem_map.mp inside
            subst candidate
            exact sameSafe alternative.child (List.mem_map_of_mem declared) false

/-- Both counters are adequate on a well-formed graph. This covers every
semantic outcome, not just successful resolutions: budget exhaustion and missing
internal declarations cannot be mistaken for a rejected union alternative. -/
theorem resolveAt_noMalformed {declarations checks keys depth rank complete identity input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (depthEnough : inputDepth input ≤ depth)
    (rankEnough : declaration.expansionRank < rank) :
    NoMalformed (resolveAt declarations checks keys depth rank complete identity input) := by
  induction depth generalizing rank complete identity input declaration with
  | zero => have positive := inputDepth_pos input; omega
  | succ depth depthIH =>
    induction rank generalizing complete identity input declaration with
    | zero => omega
    | succ rank rankIH =>
      simp only [resolveAt, found, Bind.bind, Except.bind]
      apply NoMalformed_bind
      · apply resolveBody_noMalformed depthEnough
        · intro child edge mode
          obtain ⟨childDeclaration, childFound, smaller⟩ := declaration_nonconsuming_rank wellFormed found edge
          exact rankIH childFound depthEnough (by omega)
        · intro child edge mode next smaller
          have existsChild : ∃ childDeclaration, findDeclaration declarations child = some childDeclaration := by
            rcases edge with consumed | expanded
            · exact declaration_consuming_exists wellFormed found consumed
            · obtain ⟨childDeclaration, childFound, _⟩ := declaration_nonconsuming_rank wellFormed found expanded
              exact ⟨childDeclaration, childFound⟩
          obtain ⟨childDeclaration, childFound⟩ := existsChild
          apply depthIH childFound (by omega)
          have bound := declaration_rank_le_maximum (findDeclaration_mem childFound)
          omega
      · intro resolution
        split <;> simp [NoMalformed, exceptThrow]

/-- The public resolver derives both counters from its finite input and checked
declaration table. Every present effective occurrence avoids malformed-plan
failure, whether resolution succeeds or reports a semantic rejection. -/
theorem resolve_noMalformed {declarations checks keys role effective input declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations effective = some declaration) :
    NoMalformed (resolve declarations checks keys role effective input) := by
  have valid := (validateDeclarations_iff declarations).mpr wellFormed
  simp only [resolve, valid, Bool.not_true, Bool.false_eq_true, ↓reduceIte]
  apply resolveAt_noMalformed wellFormed found (by omega)
  have bound := declaration_rank_le_maximum (findDeclaration_mem found)
  omega

end ValueContract.Candidate
