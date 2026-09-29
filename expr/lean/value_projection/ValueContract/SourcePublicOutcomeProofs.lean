import ValueContract.SourceFullOutcomeProofs
import ValueContract.SourceStabilityProofs
import ValueContract.SourceGraphBudgetProofs
import ValueContract.SourceMissingProofs

namespace ValueContract.Candidate

/-- Any adequately budgeted source worker computes exactly the unbounded
semantic outcome. The only premises concern the finite declaration graph and
structural budgets; there is no candidate-success or target assumption. -/
theorem resolveAt_semantic_outcome_iff {declarations : Declarations}
    {checks : ExternalScalarChecks} {keys : KeyCodec} {depth rank : Nat}
    {complete : Bool} {identity : Identity} {input : Input} {declaration : Declaration}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (depthEnough : inputDepth input ≤ depth) (rankEnough : declaration.expansionRank < rank)
    (result : ResolveResult) :
    resolveAt declarations checks keys depth rank complete identity input = result ↔
      SourceOutcome declarations checks keys complete identity input result := by
  constructor
  · intro computed
    exact ⟨declaration, findDeclaration_mem found, findDeclaration_identity found,
      depth, rank, depthEnough, rankEnough,
      (resolveAt_outcome_iff wellFormed _ _ _ _ _ _ _ _).mp computed⟩
  · rintro ⟨other, member, identitySame, otherDepth, otherRank, depthBound, rankBound, specified⟩
    have otherFound := identitySame ▸ findDeclaration_of_mem wellFormed member
    have declarationSame := Option.some.inj (found.symm.trans otherFound)
    subst other
    rw [resolveAt_stable wellFormed found depthEnough depthBound rankEnough rankBound]
    exact (resolveAt_outcome_iff wellFormed _ _ _ _ _ _ _ _).mpr specified

/-- Public complete-first source resolution has full semantic correspondence,
including every failure, retained union identity and exact missing-member path.
The graph/input bounds are derived internally, not supplied by a caller. -/
theorem resolve_outcome_iff {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {role : Role} {identity : Identity} {input : Input}
    {declaration : Declaration} (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration) (result : ResolveResult) :
    resolve declarations checks keys role identity input = result ↔
      SourceOutcome declarations checks keys (role == .enumMember || role == .defaultValue)
        identity input result := by
  have depthEnough : inputDepth input ≤ inputDepth input + 1 := by omega
  have rankEnough : declaration.expansionRank < maximumExpansionRank declarations + 1 := by
    have bound := declaration_rank_le_maximum (findDeclaration_mem found)
    omega
  rw [resolve_eq_resolveAt wellFormed found depthEnough rankEnough]
  exact resolveAt_semantic_outcome_iff wellFormed found depthEnough rankEnough result

/-- Adequate semantic derivations have one answer across every choice of
height/rank witness, whether acceptance, rejection or nested ambiguity. -/
theorem sourceOutcome_unique {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {complete : Bool} {identity : Identity} {input : Input}
    {declaration : Declaration} {left right : ResolveResult}
    (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration)
    (first : SourceOutcome declarations checks keys complete identity input left)
    (second : SourceOutcome declarations checks keys complete identity input right) : left = right := by
  have depthEnough : inputDepth input ≤ inputDepth input + 1 := by omega
  have rankEnough : declaration.expansionRank < maximumExpansionRank declarations + 1 := by
    have bound := declaration_rank_le_maximum (findDeclaration_mem found)
    omega
  exact ((resolveAt_semantic_outcome_iff wellFormed found depthEnough rankEnough left).mpr first).symm.trans
    ((resolveAt_semantic_outcome_iff wellFormed found depthEnough rankEnough right).mpr second)

/-- Every finite input at a valid declared root has an exact semantic outcome;
internal exhaustion is excluded even when the input is rejected. -/
theorem sourceOutcome_total {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {role : Role} {identity : Identity} {input : Input}
    {declaration : Declaration} (wellFormed : WellFormedDeclarations declarations)
    (found : findDeclaration declarations identity = some declaration) :
    ∃ result, SourceOutcome declarations checks keys (role == .enumMember || role == .defaultValue)
      identity input result ∧ NoMalformed result := by
  exact ⟨resolve declarations checks keys role identity input,
    (resolve_outcome_iff wellFormed found _).mp rfl, resolve_noMalformed wellFormed found⟩

/-- The semantic success relation itself preserves the independent typing
contract. Semantic progress therefore cannot manufacture an untyped witness. -/
theorem sourceOutcome_typed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {complete : Bool} {identity : Identity} {input : Input}
    {result : Resolution} (wellFormed : WellFormedDeclarations declarations)
    (specified : SourceOutcome declarations checks keys complete identity input (.ok result)) :
    Typed declarations checks complete identity result.value := by
  obtain ⟨_, _, _, depth, rank, _, _, indexed⟩ := specified
  exact resolveAt_typed wellFormed ((resolveAt_outcome_iff wellFormed checks keys depth rank
    complete identity input (.ok result)).mpr indexed)

/-- Completeness is observable in the exact missing-path output. This excludes
filling missing defaults or silently discarding paths to make a branch complete. -/
theorem sourceOutcome_complete_missing {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {identity : Identity} {input : Input} {result : Resolution}
    (wellFormed : WellFormedDeclarations declarations)
    (specified : SourceOutcome declarations checks keys true identity input (.ok result)) :
    result.missing = [] := by
  obtain ⟨_, _, _, depth, rank, _, _, indexed⟩ := specified
  exact resolveAt_complete_missing rfl ((resolveAt_outcome_iff wellFormed checks keys depth rank
    true identity input (.ok result)).mpr indexed)

/-- Every semantic outcome, including rejection, excludes the implementation's
internal malformed/exhaustion result at a well-formed declared source root. -/
theorem sourceOutcome_noMalformed {declarations : Declarations} {checks : ExternalScalarChecks}
    {keys : KeyCodec} {complete : Bool} {identity : Identity} {input : Input} {result : ResolveResult}
    (wellFormed : WellFormedDeclarations declarations)
    (specified : SourceOutcome declarations checks keys complete identity input result) :
    NoMalformed result := by
  obtain ⟨declaration, member, same, depth, rank, depthEnough, rankEnough, indexed⟩ := specified
  have found := same ▸ findDeclaration_of_mem wellFormed member
  rw [← (resolveAt_outcome_iff wellFormed checks keys depth rank complete identity input result).mpr indexed]
  exact resolveAt_noMalformed wellFormed found depthEnough rankEnough

end ValueContract.Candidate
