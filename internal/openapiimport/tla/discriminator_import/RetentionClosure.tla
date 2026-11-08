--------------------------- MODULE RetentionClosure ---------------------------
\* Component retention when #1005 adds normalized-only discriminator edges.
\* Raw edges are bundled source references; discriminator mapping occurrences
\* are excluded from them, and a weak child has no raw parent-to-child edge.
\* Each scenario is selected independently; nothing enumerates edge products.
EXTENDS Naturals
CONSTANTS Policy, Selected
Ops == {"o1", "o2"}
Roots(first, second) == [o \in Ops |-> IF o = "o1" THEN first ELSE second]
\* C is a weak child of P. U drops a redundant sibling whose raw $ref names the
\* blocked placeholder X, so U->X is raw-only.
WeakChild == [name |-> "weak-child",
    norm |-> {<<"P", "C">>, <<"C", "P">>, <<"C", "U">>}, weak |-> {<<"P", "C">>},
    mapping |-> {}, raw |-> {<<"C", "P">>, <<"C", "U">>, <<"U", "X">>},
    opNorm |-> Roots({"P"}, {}), opRaw |-> Roots({"P"}, {}), blocked |-> {"X"}]
\* An explicit mapping or defaultMapping target M is a strong normalized-only edge.
StrongTarget == [name |-> "strong-target",
    norm |-> {<<"P", "M">>, <<"M", "P">>, <<"M", "U">>}, weak |-> {},
    mapping |-> {<<"P", "M">>}, raw |-> {<<"M", "P">>, <<"M", "U">>, <<"U", "X">>},
    opNorm |-> Roots({"P"}, {}), opRaw |-> Roots({"P"}, {}), blocked |-> {"X"}]
\* Control: a second operation returns C directly, so C's prefix is source-walked.
RawPath == [WeakChild EXCEPT !.name = "raw-path",
    !.opNorm = Roots({"P"}, {"C"}), !.opRaw = Roots({"P"}, {"C"})]
\* Control: the unresolved reference survives normalization inside C.
Placement == [name |-> "normalized-placement",
    norm |-> {<<"P", "C">>, <<"C", "P">>, <<"C", "X">>}, weak |-> {<<"P", "C">>},
    mapping |-> {}, raw |-> {<<"C", "P">>, <<"C", "X">>},
    opNorm |-> Roots({"P"}, {}), opRaw |-> Roots({"P"}, {}), blocked |-> {"X"}]
\* C1 reaches the blocker raw-only; valid C2 has a C2/Q cycle and must survive.
Cycle == [name |-> "cycle-sibling",
    norm |-> {<<"P", "C1">>, <<"P", "C2">>, <<"C1", "P">>, <<"C1", "U">>,
              <<"C2", "P">>, <<"C2", "Q">>, <<"Q", "C2">>},
    weak |-> {<<"P", "C1">>, <<"P", "C2">>}, mapping |-> {},
    raw |-> {<<"C1", "P">>, <<"C1", "U">>, <<"U", "X">>, <<"C2", "P">>, <<"C2", "Q">>, <<"Q", "C2">>},
    opNorm |-> Roots({"P"}, {}), opRaw |-> Roots({"P"}, {}), blocked |-> {"X"}]
Scenarios == {s \in {WeakChild, StrongTarget, RawPath, Placement, Cycle} : s.name \in Selected}
VARIABLES sc, order, phase, pending, cand, qualified, op, work, retained, bodies, prefixes
vars == <<sc, order, phase, pending, cand, qualified, op, work, retained, bodies, prefixes>>
Targets(edges, c) == {e[2] : e \in {f \in edges : f[1] = c}}
RECURSIVE Close(_, _)
Close(edges, s) == LET next == s \cup UNION {Targets(edges, c) : c \in s}
                   IN IF next = s THEN s ELSE Close(edges, next)
Strong == sc.norm \ sc.weak
Candidates == {e[2] : e \in sc.weak}
Seeds == UNION {sc.opNorm[o] \cup sc.opRaw[o] : o \in Ops}
\* Contract: every retained node contributes its normalized body and its raw
\* source, a least fixed point of norm union raw. Qualification uses the same
\* closure without weak edges; only qualified weak children are projected.
OracleQualified == {c \in Candidates : Close(sc.raw \cup Strong, {c}) \cap sc.blocked = {}}
OracleRetained == Close(sc.raw \cup Strong \cup {e \in sc.weak : e[2] \in OracleQualified}, Seeds)
HintParents == {e[1] : e \in sc.weak \cup sc.mapping}
Unwitnessed(r) == {p \in HintParents \cap r : Targets(sc.weak \cup sc.mapping, p) \cap r = {}}
NormEdges == IF phase = "qualify" THEN Strong ELSE Strong \cup {e \in sc.weak : e[2] \in qualified}
\* "body-once" is frozen R4/R9: only operation prefixes are source-walked.
\* "projection-only" source-walks retained components in the projection walk
\* but not in qualification. "per-edge" source-walks only discriminator-edge
\* targets. "closure" is the central owner: every retained component, once.
DiscriminatorTargets == {e[2] : e \in sc.weak \cup sc.mapping}
SourceWalks(c) == \/ (Policy = "closure")
                  \/ (Policy = "projection-only" /\ phase # "qualify")
                  \/ (Policy = "per-edge" /\ c \in DiscriminatorTargets)
OwnerSources == (Policy = "closure") \/ (Policy = "projection-only" /\ phase # "qualify")
NormItems(s) == {<<"norm", c>> : c \in s}
RawItems(s) == {<<"raw", c>> : c \in s}
OpRawItems(o) == IF OwnerSources THEN NormItems(sc.opRaw[o]) ELSE RawItems(sc.opRaw[o])
Init == /\ sc \in Scenarios
        /\ order \in {<<"o1", "o2">>, <<"o2", "o1">>}
        /\ phase = "qualify"
        /\ pending = Candidates
        /\ cand = "none"
        /\ qualified = {}
        /\ op = 1
        /\ work = {}
        /\ retained = {}
        /\ bodies = {}
        /\ prefixes = {}
\* retainComponent: retain the name, walk its normalized body once and, when the
\* policy source-walks it, its raw prefix once. Raw targets retain the same way.
VisitNorm(item) ==
    LET c == item[2]
        body == c \notin bodies
        source == SourceWalks(c) /\ c \notin prefixes
    IN /\ retained' = retained \cup {c}
       /\ bodies' = IF body THEN bodies \cup {c} ELSE bodies
       /\ prefixes' = IF source THEN prefixes \cup {c} ELSE prefixes
       /\ work' = (work \ {item}) \cup (IF body THEN NormItems(Targets(NormEdges, c)) ELSE {})
                  \cup (IF source THEN NormItems(Targets(sc.raw, c)) ELSE {})
\* The R4/R9 operation-prefix source walk.
VisitRaw(item) ==
    LET c == item[2] IN
    IF c \in prefixes
    THEN work' = work \ {item} /\ UNCHANGED <<retained, bodies, prefixes>>
    ELSE /\ prefixes' = prefixes \cup {c}
         /\ retained' = retained \cup {c}
         /\ work' = (work \ {item}) \cup RawItems(Targets(sc.raw, c)) \cup NormItems({c})
         /\ UNCHANGED bodies
Visit == /\ work # {}
         /\ \E item \in work: IF item[1] = "norm" THEN VisitNorm(item) ELSE VisitRaw(item)
         /\ UNCHANGED <<sc, order, phase, pending, cand, qualified, op>>
\* Each candidate is qualified by a fresh pruner seeded with its declaration.
StartQualify == /\ phase = "qualify" /\ cand = "none" /\ work = {}
                /\ \E c \in pending: cand' = c /\ pending' = pending \ {c} /\ work' = NormItems({c})
                /\ retained' = {} /\ bodies' = {} /\ prefixes' = {}
                /\ UNCHANGED <<sc, order, phase, qualified, op>>
FinishQualify == /\ phase = "qualify" /\ cand # "none" /\ work = {}
                 /\ qualified' = IF retained \cap sc.blocked = {} THEN qualified \cup {cand} ELSE qualified
                 /\ cand' = "none"
                 /\ UNCHANGED <<sc, order, phase, pending, op, work, retained, bodies, prefixes>>
StartProject == /\ phase = "qualify" /\ cand = "none" /\ pending = {} /\ work = {}
                /\ phase' = "norm"
                /\ work' = NormItems(sc.opNorm[order[1]])
                /\ retained' = {} /\ bodies' = {} /\ prefixes' = {}
                /\ UNCHANGED <<sc, order, pending, cand, qualified, op>>
\* Per operation: normalized walk, then its source walk. The old walk used a
\* fresh prefix map per operation; owner-tracked sources are per pruner.
Advance == /\ phase \in {"norm", "raw"} /\ work = {}
           /\ \/ /\ phase = "norm" /\ phase' = "raw" /\ work' = OpRawItems(order[op])
                 /\ prefixes' = IF OwnerSources THEN prefixes ELSE {}
                 /\ UNCHANGED op
              \/ /\ phase = "raw" /\ op = 1 /\ op' = 2 /\ phase' = "norm"
                 /\ work' = NormItems(sc.opNorm[order[2]]) /\ UNCHANGED prefixes
              \/ /\ phase = "raw" /\ op = 2 /\ phase' = "done"
                 /\ UNCHANGED <<op, work, prefixes>>
           /\ UNCHANGED <<sc, order, pending, cand, qualified, retained, bodies>>
Next == StartQualify \/ FinishQualify \/ StartProject \/ Visit \/ Advance
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Done == phase = "done"
QualificationExact == Done => qualified = OracleQualified
ExactClosure == Done => retained = OracleRetained
BlockerOutcome == Done => retained \cap sc.blocked = OracleRetained \cap sc.blocked
HintOutcome == Done => Unwitnessed(retained) = Unwitnessed(OracleRetained)
OptionalCannotPoison == Done => \A c \in qualified: Close(sc.raw \cup Strong, {c}) \cap sc.blocked = {}
StrongBlockersRemain == Done => Close(sc.raw \cup Strong, Seeds) \cap sc.blocked \subseteq retained
WalkComplete == Done => bodies = retained /\ retained \subseteq prefixes
Termination == <>Done
=============================================================================
