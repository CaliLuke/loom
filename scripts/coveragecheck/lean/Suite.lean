-- Pure ownership model for coveragecheck's distributed suite.
-- CI supplies a fixed package inventory and immutable source revision.
def belongs (owner : Nat → Nat) (lane package : Nat) : Prop := owner package = lane

theorem every_package_has_owner (owner : Nat → Nat) (package : Nat) :
    belongs owner (owner package) package := rfl

theorem no_duplicate_owner (owner : Nat → Nat) (package a b : Nat)
    (ha : belongs owner a package) (hb : belongs owner b package) : a = b := by
  exact ha.symm.trans hb

-- Execution happens once in the owner lane. Race and coverage are observers
-- of that execution, rather than independent executions of the same tests.
def accepted (expected received : Nat → Prop) (sameSource passed : Nat → Prop) : Prop :=
  (∀ lane, expected lane ↔ received lane) ∧
  (∀ lane, received lane → sameSource lane ∧ passed lane)

theorem accepted_is_complete (expected received sameSource passed : Nat → Prop)
    (h : accepted expected received sameSource passed) (lane : Nat)
    (required : expected lane) : received lane ∧ sameSource lane ∧ passed lane := by
  have present := (h.1 lane).mp required
  exact ⟨present, h.2 lane present⟩

-- The baseline uses only configured consumers. Other tests still execute,
-- but their observations cannot raise the protected boundary's coverage.
def covered (selected observed : Nat → Prop) : Prop := ∃ p, selected p ∧ observed p

theorem excluded_consumer_cannot_inflate (selected before after : Nat → Prop)
    (same : ∀ p, selected p → (before p ↔ after p)) :
    covered selected before ↔ covered selected after := by
  constructor
  · rintro ⟨p, hp, seen⟩
    exact ⟨p, hp, (same p hp).mp seen⟩
  · rintro ⟨p, hp, seen⟩
    exact ⟨p, hp, (same p hp).mpr seen⟩
