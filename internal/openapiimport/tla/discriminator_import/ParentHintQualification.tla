--------------------- MODULE ParentHintQualification ---------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES base, needs, inline, strong, eligible, phase, retained, blocked, selected, rootQ
vars == <<base, needs, inline, strong, eligible, phase, retained, blocked, selected, rootQ>>
Children == {1, 2, 3}
Parents == {"P", "Q"}
Owner(c) == IF c = 3 THEN "Q" ELSE "P"
OperationInline == IF selected THEN {"Q"} ELSE {}
\* A selected operation may also reference Q strongly without supplying a child.
OperationRoots == OperationInline \cup (IF rootQ THEN {"Q"} ELSE {})
ParentsVisited(nodes) == {"P"} \cup OperationRoots \cup
    UNION {needs[c] \cup inline[c] \cup {Owner(c)} : c \in nodes}
RECURSIVE Reach(_, _, _)
Reach(nodes, candidates, depth) ==
    IF depth = 0 THEN nodes
    ELSE LET prior == Reach(nodes, candidates, depth - 1)
         IN prior \cup {c \in candidates: Owner(c) \in ParentsVisited(prior)}
Reachable(c, candidates) == Reach({c} \cup strong, candidates, 3)
Witnessed(c, p, candidates) == p \in OperationInline \/
    (\E child \in Reachable(c, candidates): Owner(child) = p \/ p \in inline[child])
\* Hints in the selected strong graph exist without any weak child. A weak child
\* cannot make them unwitnessed; it is judged only on obligations it introduces.
RootOwned == ParentsVisited(strong)
Introduced(c) == needs[c] \ RootOwned
Qualified(c, candidates) == \A p \in Introduced(c): Witnessed(c, p, candidates)
\* Base plans are immutable. Reachability belongs to the selected projection.
\* An unselected operation has an inline Q child but cannot be a witness.
ValidSets == {candidate \in SUBSET base:
                \A c \in candidate: Qualified(c, candidate)}
Viable == UNION ValidSets
\* Only reachable children are emitted; unreachable eligibility is unobservable.
Emittable == Viable \cap Reach(strong, Viable, 3)
\* Final coherence and partial omission: visited hints without a retained witness.
Unwitnessed(nodes) == {p \in ParentsVisited(nodes):
    ~(p \in OperationInline \/ \E child \in nodes: Owner(child) = p \/ p \in inline[child])}
\* One walk from the selected roots under the current weak edges serves every
\* reached candidate in a round.
Shared == Reach(strong, eligible, 3)
SharedWitnessed(p) == p \in OperationInline \/
    (\E child \in Shared: Owner(child) = p \/ p \in inline[child])
Exempt == CASE Mode = "weak-owned" -> ParentsVisited(Shared)
            [] Mode = "fixed-point" -> RootOwned
            [] OTHER -> {}
CandidateWitness(c, p) == p \in Exempt \/
    CASE Mode \in {"roots-only", "roots-only-sibling"} -> (p \in inline[c] \/
                                (\E child \in eligible: Owner(child) = p))
      [] Mode = "unselected-witness" -> (p = "Q" \/ SharedWitnessed(p))
      [] OTHER -> SharedWitnessed(p)
Bad == {c \in eligible: c \notin Shared \/ ~\A p \in needs[c]: CandidateWitness(c, p)}
Init == /\ base \in SUBSET Children
        /\ needs \in [Children -> SUBSET Parents]
        /\ inline \in {[c \in Children |-> {}],
                         [c \in Children |-> IF c = 1 THEN {"Q"} ELSE {}],
                         [c \in Children |-> IF c = 2 THEN {"Q"} ELSE {}]}
        /\ strong \in SUBSET Children
        /\ selected \in (IF Mode = "roots-only-sibling" THEN {FALSE} ELSE BOOLEAN)
        /\ rootQ \in (IF Mode = "roots-only-sibling" THEN {FALSE} ELSE BOOLEAN)
        /\ (Mode # "roots-only-sibling" \/
             (base = {1, 2} /\ strong = {} /\
              inline = [c \in Children |-> IF c = 2 THEN {"Q"} ELSE {}] /\
              needs = [c \in Children |-> IF c = 1 THEN {"Q"} ELSE {}]))
        /\ eligible = IF Mode = "empty-seed" THEN {} ELSE base
        /\ phase = "qualify"
        /\ retained = {}
        /\ blocked = FALSE
\* Any failing child can be removed next. Convergence must be order independent.
Remove == /\ phase = "qualify" /\ Mode # "one-pass" /\ Bad # {}
          /\ \E child \in Bad: eligible' = eligible \ {child}
          /\ UNCHANGED <<base, needs, inline, strong, phase, retained, blocked, selected, rootQ>>
Publish == /\ phase = "qualify" /\ (Mode = "one-pass" \/ Bad = {})
           /\ retained' = IF Mode = "weaken-strong" THEN Reach({}, eligible, 3) ELSE Reach(strong, eligible, 3)
           /\ blocked' = ((retained' \ base) # {} \/ Unwitnessed(retained') # {})
           /\ phase' = "done"
           /\ UNCHANGED <<base, needs, inline, strong, eligible, selected, rootQ>>
Next == Remove \/ Publish
Spec == Init /\ [][Next]_vars
ClosureRepresentable == phase = "done" => eligible \subseteq Viable
MaximalAndOrderIndependent == phase = "done" => eligible = Emittable
StrongRetained == phase = "done" => strong \subseteq retained
StrongFailuresRemain == phase = "done" /\
    ((strong \ base) \cup {c \in strong: ~\A p \in needs[c]: Witnessed(c, p, Viable)}) # {} => blocked
OptionalCannotPoison == phase = "done" /\ strong \subseteq base /\
                       Unwitnessed(Reach(strong, Viable, 3)) = {} => ~blocked
WeakIntroducesNoOmission == phase = "done" => Unwitnessed(retained) \subseteq RootOwned
=============================================================================
