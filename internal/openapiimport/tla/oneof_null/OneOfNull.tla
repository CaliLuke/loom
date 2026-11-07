--------------------------- MODULE OneOfNull ---------------------------
\* Null membership in untagged oneOf normalization (#1014 section 6) with
\* intrinsic Any and nested unions. Source trees are OpenAPI oneOf schemas:
\* leaves admit null not at all ("e"), at this occurrence ("o", type: [T, null])
\* or through a declared nullable component ("d"); Any admits null; a nested
\* union admits null when exactly one of its alternatives does, bare
\* {type: null} branches included. Normalization runs bottom-up: inner unions
\* first, then the outer union is processed branch by branch.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode

Leaf(n) == [kind |-> "leaf", null |-> n, bare |-> 0, flag |-> FALSE, kids |-> <<>>]
AnyT == [Leaf("e") EXCEPT !.kind = "any"]
U(a, b, bare) == [kind |-> "union", null |-> "e", bare |-> bare, flag |-> FALSE, kids |-> <<a, b>>]
Level0 == {Leaf("e"), Leaf("o"), Leaf("d"), AnyT}
Level1 == {U(a, b, n) : a \in Level0, b \in Level0, n \in 0..1}
Level2 == {U(a, b, n) : a \in Level1, b \in Level0, n \in 0..1}

\* Source semantics: null counted like every value.
RECURSIVE SrcMem(_)
SrcMem(t) == CASE t.kind = "leaf" -> t.null # "e"
             [] t.kind = "any" -> TRUE
             [] OTHER -> Cardinality({j \in 1..Len(t.kids) : SrcMem(t.kids[j])}) + t.bare = 1

\* Loom representation semantics: occurrence flag accepts null; otherwise
\* exact-one over branches, each counting its own null admission.
RECURSIVE RepMem(_)
Members(t) == {j \in 1..Len(t.kids) : RepMem(t.kids[j])}
RepMem(t) == CASE t.kind = "leaf" -> t.null # "e"
             [] t.kind = "any" -> TRUE
             [] OTHER -> t.flag \/ Cardinality(Members(t)) = 1

\* Null kind of a representation branch: "o" when its null can be removed
\* without touching its non-null domain, "d" when it cannot, "e" for none.
RECURSIVE Kind(_)
Kind(t) == CASE t.kind = "leaf" -> t.null
             [] t.kind = "any" -> "d"
             [] OTHER -> LET c == Cardinality(Members(t)) IN
                           IF t.flag
                              THEN IF c = 1 THEN Kind(t.kids[CHOOSE j \in Members(t) : TRUE]) ELSE "o"
                              ELSE IF c = 1 THEN Kind(t.kids[CHOOSE j \in Members(t) : TRUE]) ELSE "e"
\* Rejected classifiers: leaf flags only, or union occurrence flag only.
ClassKind(t) == CASE Mode = "flag-only" ->
                       IF t.kind = "leaf" THEN t.null ELSE IF t.flag THEN "o" ELSE "e"
                  [] Mode = "shallow" ->
                       IF t.kind = "union" THEN (IF t.flag THEN "o" ELSE "e") ELSE Kind(t)
                  [] OTHER -> Kind(t)

RECURSIVE Strip(_)
Strip(t) == CASE t.kind = "leaf" -> IF t.null = "o" THEN Leaf("e") ELSE t
              [] t.kind = "union" ->
                   IF Cardinality(Members(t)) = 1
                      THEN LET j == CHOOSE j \in Members(t) : TRUE
                           IN [t EXCEPT !.flag = FALSE, !.kids[j] = Strip(t.kids[j])]
                      ELSE [t EXCEPT !.flag = FALSE]
              [] OTHER -> t

\* The bare-null rule over already normalized branches (same as the Lean model).
Rule(kids, bare) ==
  LET ks == [j \in 1..Len(kids) |-> ClassKind(kids[j])]
      k == Cardinality({j \in 1..Len(kids) : ks[j] # "e"})
      occ == {j \in 1..Len(kids) : ks[j] = "o"}
      base == [kind |-> "union", null |-> "e", bare |-> 0, flag |-> FALSE, kids |-> kids]
  IN CASE bare = 0 -> [ok |-> TRUE, rep |-> base]
       [] k = 0 -> [ok |-> TRUE, rep |-> [base EXCEPT !.flag = (bare = 1)]]
       [] k >= 2 -> [ok |-> TRUE, rep |-> base]
       [] occ # {} -> LET j == CHOOSE j \in occ : TRUE
                      IN [ok |-> TRUE, rep |-> [base EXCEPT !.kids[j] = Strip(kids[j])]]
       [] OTHER -> [ok |-> FALSE, rep |-> base]

RECURSIVE Norm(_)
Norm(t) == IF t.kind # "union" THEN [ok |-> TRUE, rep |-> t]
           ELSE LET a == Norm(t.kids[1]) b == Norm(t.kids[2])
                IN IF a.ok /\ b.ok THEN Rule(<<a.rep, b.rep>>, t.bare)
                   ELSE [ok |-> FALSE, rep |-> t]

VARIABLES src, inner, i, phase, result
vars == <<src, inner, i, phase, result>>

\* The outer oneOf: one tree of depth <= 2, one of depth <= 1, 0..2 bare nulls.
Init == /\ \E a \in Level0 \cup Level1 \cup Level2, b \in Level0 \cup Level1, n \in 0..2 :
             src = U(a, b, n)
        /\ inner = <<>> /\ i = 1 /\ phase = "branches"
        /\ result = [ok |-> FALSE, rep |-> Leaf("e")]

\* Normalize each outer branch (its subtree bottom-up), one branch per step.
Branch == /\ phase = "branches" /\ i <= 2
          /\ inner' = Append(inner, Norm(src.kids[i]))
          /\ i' = i + 1
          /\ UNCHANGED <<src, phase, result>>

Decide == /\ phase = "branches" /\ i > 2
          /\ result' = IF inner[1].ok /\ inner[2].ok
                          THEN Rule(<<inner[1].rep, inner[2].rep>>, src.bare)
                          ELSE [ok |-> FALSE, rep |-> Leaf("e")]
          /\ phase' = "done"
          /\ UNCHANGED <<src, inner, i>>

Next == Branch \/ Decide
Spec == Init /\ [][Next]_vars

\* An admitted representation accepts null exactly when the source does.
NullAgreement == phase = "done" /\ result.ok => RepMem(result.rep) = SrcMem(src)
\* Rejection is limited to a declared null (component, Any or nested) beside a
\* bare null, at some level.
RECURSIVE Blocked(_)
Blocked(t) == t.kind = "union" /\
              (\/ \E j \in 1..2 : Blocked(t.kids[j])
               \/ LET kids == <<Norm(t.kids[1]).rep, Norm(t.kids[2]).rep>>
                      ks == {j \in 1..2 : Kind(kids[j]) # "e"}
                  IN t.bare >= 1 /\ Cardinality(ks) = 1 /\
                     Kind(kids[CHOOSE j \in ks : TRUE]) = "d")
RejectionIsNecessary == phase = "done" /\ ~result.ok => Blocked(src)
=============================================================================
