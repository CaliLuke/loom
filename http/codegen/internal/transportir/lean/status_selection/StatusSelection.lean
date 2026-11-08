import Std
namespace Loom.HTTP.StatusSelection

/-- OpenAPI response keys: an exact code, a class digit (`4XX`) or `default`. -/
inductive Key where
  | exact (code : Nat)
  | cls (digit : Nat)
  | fallback
  deriving DecidableEq, Repr

def classOf (c : Nat) : Nat := c / 100
def isFinal (c : Nat) : Prop := 200 ≤ c ∧ c ≤ 599
def isInterim (c : Nat) : Prop := 100 ≤ c ∧ c ≤ 199 ∧ c ≠ 101
instance (c : Nat) : Decidable (isFinal c) := by unfold isFinal; infer_instance
instance (c : Nat) : Decidable (isInterim c) := by unfold isInterim; infer_instance

def specificity : Key → Nat
  | .exact _ => 2
  | .cls _ => 1
  | .fallback => 0

/-- Final-phase coverage. `default` covers only final statuses. -/
def coversFinal : Key → Nat → Prop
  | .exact k, c => k = c ∧ isFinal c
  | .cls n, c => classOf c = n ∧ isFinal c
  | .fallback, c => isFinal c

/-- Design selection: membership tests in fixed precedence. -/
def selectFinal (keys : List Key) (c : Nat) : Option Key :=
  if isFinal c then
    if Key.exact c ∈ keys then some (.exact c)
    else if Key.cls (classOf c) ∈ keys then some (.cls (classOf c))
    else if Key.fallback ∈ keys then some .fallback
    else none
  else none

/-- Interim selection for observed informational responses; 101 never selects. -/
def selectInterim (keys : List Key) (c : Nat) : Option Key :=
  if isInterim c then
    if Key.exact c ∈ keys then some (.exact c)
    else if Key.cls 1 ∈ keys then some (.cls 1)
    else none
  else none

/-- Server runtime domain written as the design formula, independently of selection. -/
def inDomain (keys : List Key) : Key → Nat → Prop
  | .exact k, c => isFinal c ∧ k = c
  | .cls n, c => isFinal c ∧ classOf c = n ∧ Key.exact c ∉ keys
  | .fallback, c => isFinal c ∧ Key.exact c ∉ keys ∧ Key.cls (classOf c) ∉ keys

/-- Rejected candidate: first covering key in declaration order. -/
def firstCover : List Key → Nat → Option Key
  | [], _ => none
  | k :: ks, c =>
    match k with
    | .exact x => if x = c ∧ isFinal c then some k else firstCover ks c
    | .cls n => if classOf c = n ∧ isFinal c then some k else firstCover ks c
    | .fallback => if isFinal c then some k else firstCover ks c

theorem select_membership_invariant (keys keys' : List Key) (c : Nat)
    (h : ∀ k, k ∈ keys ↔ k ∈ keys') : selectFinal keys c = selectFinal keys' c := by
  simp only [selectFinal, h]

theorem select_perm_invariant (keys keys' : List Key) (c : Nat)
    (p : keys.Perm keys') : selectFinal keys c = selectFinal keys' c :=
  select_membership_invariant keys keys' c (fun _ => p.mem_iff)

theorem select_covers (keys : List Key) (c : Nat) (k : Key)
    (h : selectFinal keys c = some k) : k ∈ keys ∧ coversFinal k c := by
  unfold selectFinal at h
  by_cases hf : isFinal c
  · simp only [hf, ite_true] at h
    by_cases he : Key.exact c ∈ keys
    · simp only [he, ite_true, Option.some.injEq] at h
      subst h; exact ⟨he, rfl, hf⟩
    · simp only [he, ite_false] at h
      by_cases hc : Key.cls (classOf c) ∈ keys
      · simp only [hc, ite_true, Option.some.injEq] at h
        subst h; exact ⟨hc, rfl, hf⟩
      · simp only [hc, ite_false] at h
        by_cases hd : Key.fallback ∈ keys
        · simp only [hd, ite_true, Option.some.injEq] at h
          subst h; exact ⟨hd, hf⟩
        · simp [hd] at h
  · simp [hf] at h

theorem select_most_specific (keys : List Key) (c : Nat) (k j : Key)
    (h : selectFinal keys c = some k) (hj : j ∈ keys) (hc : coversFinal j c) :
    specificity j ≤ specificity k := by
  have hf : isFinal c := by
    cases j with
    | exact x => exact hc.2
    | cls n => exact hc.2
    | fallback => exact hc
  unfold selectFinal at h
  simp only [hf, ite_true] at h
  cases j with
  | exact x =>
    have hx : x = c := hc.1
    subst hx
    simp only [hj, ite_true, Option.some.injEq] at h
    subst h; exact Nat.le_refl _
  | cls n =>
    have hn : classOf c = n := hc.1
    subst hn
    by_cases he : Key.exact c ∈ keys
    · simp only [he, ite_true, Option.some.injEq] at h
      subst h; simp [specificity]
    · simp only [he, ite_false, hj, ite_true, Option.some.injEq] at h
      subst h; exact Nat.le_refl _
  | fallback =>
    cases k <;> simp [specificity]

theorem domain_iff_select (keys : List Key) (k : Key) (c : Nat) (hk : k ∈ keys) :
    inDomain keys k c ↔ selectFinal keys c = some k := by
  cases k with
  | exact x =>
    constructor
    · intro ⟨hf, hx⟩
      subst hx
      simp [selectFinal, hf, hk]
    · intro h
      have := select_covers keys c (.exact x) h
      exact ⟨this.2.2, this.2.1⟩
  | cls n =>
    constructor
    · intro ⟨hf, hn, he⟩
      subst hn
      simp [selectFinal, hf, he, hk]
    · intro h
      have cov := select_covers keys c (.cls n) h
      refine ⟨cov.2.2, cov.2.1, ?_⟩
      intro he
      simp [selectFinal, cov.2.2, he] at h
  | fallback =>
    constructor
    · intro ⟨hf, he, hc⟩
      simp [selectFinal, hf, he, hc, hk]
    · intro h
      have hf : isFinal c := (select_covers keys c .fallback h).2
      refine ⟨hf, ?_, ?_⟩
      · intro he
        simp [selectFinal, hf, he] at h
      · intro hc
        by_cases he : Key.exact c ∈ keys
        · simp [selectFinal, hf, he] at h
        · simp [selectFinal, hf, he, hc] at h

theorem phases_disjoint (keys : List Key) (c : Nat) :
    selectFinal keys c = none ∨ selectInterim keys c = none := by
  by_cases hf : isFinal c
  · right
    have : ¬ isInterim c := by
      intro hi; unfold isFinal at hf; unfold isInterim at hi; omega
    simp [selectInterim, this]
  · left
    simp [selectFinal, hf]

theorem switching_protocols_never_interim (keys : List Key) :
    selectInterim keys 101 = none ∧ selectFinal keys 101 = none := by
  constructor
  · simp [selectInterim, isInterim]
  · simp [selectFinal, isFinal]

theorem declaration_order_counterexample :
    firstCover [.fallback, .cls 2] 200 = some .fallback ∧
    selectFinal [.fallback, .cls 2] 200 = some (.cls 2) := by
  decide

#print axioms select_membership_invariant
#print axioms select_perm_invariant
#print axioms select_covers
#print axioms select_most_specific
#print axioms domain_iff_select
#print axioms phases_disjoint
#print axioms switching_protocols_never_interim
#print axioms declaration_order_counterexample
end Loom.HTTP.StatusSelection
