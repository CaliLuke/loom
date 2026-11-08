--------------------------- MODULE SourceRetention ---------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
Ops == {"o1", "o2"}
Roots == {"X", "Y"}
\* Includes the X/Y cycle. Normalized and raw edges are chosen independently:
\* promoted declarations are normalized-only, dropped siblings raw-only.
Edges == {<<"X", "Y">>, <<"Y", "Z">>, <<"Y", "X">>}
VARIABLES norm, raw, opNorm, opRaw, order, op, stage, work, retained, bodies, prefixes
vars == <<norm, raw, opNorm, opRaw, order, op, stage, work, retained, bodies, prefixes>>
Targets(edges, c) == {e[2] : e \in {f \in edges : f[1] = c}}
RECURSIVE Close(_, _)
Close(edges, s) == LET next == s \cup UNION {Targets(edges, c) : c \in s}
                   IN IF next = s THEN s ELSE Close(edges, next)
\* Source recursion follows raw edges from operation raw roots; every retained
\* name also contributes its normalized body.
Expected == Close(norm, Close(raw, UNION {opRaw[o] : o \in Ops}) \cup UNION {opNorm[o] : o \in Ops})
NormItems(s) == {<<"norm", c>> : c \in s}
RawItems(s) == {<<"raw", c>> : c \in s}
Init == /\ norm \in SUBSET Edges
        /\ raw \in SUBSET Edges
        /\ opNorm \in [Ops -> SUBSET Roots]
        /\ opRaw \in [Ops -> SUBSET Roots]
        /\ order \in {<<"o1", "o2">>, <<"o2", "o1">>}
        /\ op = 1
        /\ stage = "norm"
        /\ work = NormItems(opNorm[order[1]])
        /\ retained = {}
        /\ bodies = {}
        /\ prefixes = {}
\* A normalized Ref visit. "name-guard" is the previous pruner: retention guards
\* the body walk. The other policies guard the body by its own visited set.
VisitNorm(item) ==
    LET c == item[2] IN
    /\ retained' = retained \cup {c}
    /\ IF (IF Policy = "name-guard" THEN c \in retained ELSE c \in bodies)
       THEN work' = work \ {item} /\ UNCHANGED bodies
       ELSE /\ bodies' = bodies \cup {c}
            /\ work' = (work \ {item}) \cup NormItems(Targets(norm, c))
    /\ UNCHANGED prefixes
\* A raw source reference: retain the name and recurse into its source prefix.
\* "body-once" also visits the retained name's normalized body.
VisitRaw(item) ==
    LET c == item[2] IN
    IF c \in prefixes
    THEN work' = work \ {item} /\ UNCHANGED <<retained, bodies, prefixes>>
    ELSE /\ prefixes' = prefixes \cup {c}
         /\ retained' = retained \cup {c}
         /\ work' = (work \ {item}) \cup RawItems(Targets(raw, c)) \cup
                    (IF Policy = "body-once" THEN NormItems({c}) ELSE {})
         /\ UNCHANGED bodies
Visit == /\ stage \in {"norm", "raw"}
         /\ \E item \in work: IF item[1] = "norm" THEN VisitNorm(item) ELSE VisitRaw(item)
         /\ UNCHANGED <<norm, raw, opNorm, opRaw, order, op, stage>>
\* Per operation: normalized walk, then its source walk with fresh prefixes.
Advance == /\ work = {}
           /\ \/ stage = "norm" /\ stage' = "raw" /\ work' = RawItems(opRaw[order[op]])
                 /\ prefixes' = {} /\ UNCHANGED op
              \/ stage = "raw" /\ op = 1 /\ op' = 2 /\ stage' = "norm"
                 /\ work' = NormItems(opNorm[order[2]]) /\ UNCHANGED prefixes
              \/ stage = "raw" /\ op = 2 /\ stage' = "done" /\ UNCHANGED <<op, work, prefixes>>
           /\ UNCHANGED <<norm, raw, opNorm, opRaw, order, retained, bodies>>
Next == Visit \/ Advance
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
BodyComplete == stage = "done" => \A c \in retained: Targets(norm, c) \subseteq retained
ExactClosure == stage = "done" => retained = Expected
RetainedBodiesWalked == stage = "done" => bodies = retained
Termination == <>(stage = "done")
=============================================================================
