------------------------ MODULE UntaggedClosure ------------------------
\* `unevaluatedProperties: false` beside an untagged oneOf (#1014 section 6,
\* #1005 closure). JSON Schema 2020-12 evaluates oneOf first; only the passing
\* branch contributes annotations, and the closure then rejects instance
\* members that annotation set does not contain. The checked algorithm counts
\* the original branches, then closes over the selected branch's evaluated
\* members. Rejected candidates close branches before counting, or use
\* declared field-name lists instead of evaluation annotations.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Profile, Mode

M == {"kind", "x", "y", "z"}
Obj(decl, req, closed) == [kind |-> "object", decl |-> decl, req |-> req, closed |-> closed,
                           additional |-> FALSE, inner |-> <<>>]
\* Declared members plus additionalProperties: true (representation owned by #1013).
ObjAdd(decl, req) == [Obj(decl, req, FALSE) EXCEPT !.additional = TRUE]
Map == [Obj({}, {}, FALSE) EXCEPT !.kind = "map"]
AnyB == [Obj({}, {}, FALSE) EXCEPT !.kind = "any"]
Scalar == [Obj({}, {}, FALSE) EXCEPT !.kind = "scalar"]
Union(a, b) == [Obj({}, {}, FALSE) EXCEPT !.kind = "union", !.inner = <<a, b>>]

Branches ==
  CASE Profile = "open-pair" ->        \* review counterexample: A requires x; B optional y
         <<Obj({"x"}, {"x"}, FALSE), Obj({"y"}, {}, FALSE)>>
    [] Profile = "discriminated" ->    \* both declare the discriminating property
         <<Obj({"kind", "x"}, {"kind"}, FALSE), Obj({"kind", "y"}, {"kind"}, FALSE)>>
    [] Profile = "map" ->
         <<Map, Scalar>>
    [] Profile = "any" ->
         <<AnyB, Obj({"x"}, {"x"}, FALSE)>>
    [] Profile = "nested" ->
         <<Union(Obj({"x"}, {"x"}, FALSE), Obj({"y"}, {"y"}, TRUE)), Scalar>>
    [] Profile = "additional" ->
         <<ObjAdd({"x"}, {"x"}), Scalar>>
    [] Profile = "singleton-scalar" -> <<Scalar>>
    [] Profile = "singleton-object" -> <<Obj({"x"}, {"x"}, FALSE)>>
    [] Profile = "singleton-any" -> <<AnyB>>
    [] Profile = "singleton-map" -> <<Map>>
    [] Profile = "singleton-nested" -> <<Union(Obj({"x"}, {"x"}, FALSE), Scalar)>>

\* An instance is a scalar or an object with a member set.
Instances == {[obj |-> FALSE, mem |-> {}]} \cup {[obj |-> TRUE, mem |-> s] : s \in SUBSET M}

RECURSIVE Accepts(_, _), Matches(_, _)
Matches(seq, v) == {j \in 1..Len(seq) : Accepts(seq[j], v)}
Accepts(b, v) ==
  CASE b.kind = "object" -> v.obj /\ b.req \subseteq v.mem /\ (b.closed => v.mem \subseteq b.decl)
    [] b.kind = "map" -> v.obj
    [] b.kind = "any" -> TRUE
    [] b.kind = "scalar" -> ~v.obj
    [] b.kind = "union" -> Cardinality(Matches(b.inner, v)) = 1

\* Evaluated-property annotations of an accepting branch at this location.
RECURSIVE Evaluated(_, _)
Evaluated(b, v) ==
  CASE b.kind = "object" -> IF b.additional \/ b.closed THEN v.mem ELSE v.mem \cap b.decl
    [] b.kind = "map" -> v.mem
    [] b.kind \in {"any", "scalar"} -> {}
    [] b.kind = "union" -> LET j == CHOOSE j \in Matches(b.inner, v) : TRUE
                           IN Evaluated(b.inner[j], v)

\* Specification oracle: exactly one original branch, then closure.
Source(v) == Cardinality(Matches(Branches, v)) = 1 /\
             (v.obj => v.mem \subseteq Evaluated(Branches[CHOOSE j \in Matches(Branches, v) : TRUE], v))

\* Rejected candidate: close every branch root before counting.
RECURSIVE CloseFirst(_)
CloseFirst(b) ==
  CASE b.kind = "object" -> [b EXCEPT !.closed = ~b.additional]
    [] b.kind = "any" -> [Obj({}, {}, TRUE) EXCEPT !.inner = <<>>]
    [] b.kind = "union" -> [b EXCEPT !.inner = <<CloseFirst(b.inner[1]), CloseFirst(b.inner[2])>>]
    [] OTHER -> b
\* Rejected candidate: declared field-name lists as the evaluated set.
RECURSIVE Names(_)
Names(b) == IF b.kind = "union" THEN Names(b.inner[1]) \cup Names(b.inner[2]) ELSE b.decl

Candidate(j) == IF Mode = "close-first" THEN CloseFirst(Branches[j]) ELSE Branches[j]
Closure(j, v) == IF Mode = "names-list" THEN v.mem \subseteq Names(Branches[j])
                 ELSE IF Mode \in {"close-first", "ignored"} THEN TRUE
                 ELSE v.mem \subseteq Evaluated(Branches[j], v)

VARIABLES v, i, count, last, phase, accepted
vars == <<v, i, count, last, phase, accepted>>

Init == /\ v \in Instances /\ i = 1 /\ count = 0 /\ last = 0
        /\ phase = "count" /\ accepted = FALSE

CountStep == /\ phase = "count" /\ i <= Len(Branches)
             /\ IF Accepts(Candidate(i), v) THEN count' = count + 1 /\ last' = i
                                            ELSE UNCHANGED <<count, last>>
             /\ i' = i + 1
             /\ UNCHANGED <<v, phase, accepted>>

Close == /\ phase = "count" /\ i > Len(Branches)
         /\ accepted' = (count = 1 /\ (Mode # "minimum-two" \/ Len(Branches) >= 2) /\ (v.obj => Closure(last, v)))
         /\ phase' = "done"
         /\ UNCHANGED <<v, i, count, last>>

Next == CountStep \/ Close
Spec == Init /\ [][Next]_vars

Agreement == phase = "done" => accepted = Source(v)
=============================================================================
