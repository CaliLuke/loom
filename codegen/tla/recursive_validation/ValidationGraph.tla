----------------------- MODULE ValidationGraph -----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES mutual, fieldFirst, pending, emitted, phase
vars == <<mutual, fieldFirst, pending, emitted, phase>>
Types == {"A", "B"}
NextType(t) == IF mutual THEN IF t = "A" THEN "B" ELSE "A" ELSE t
Reachable == IF mutual THEN Types ELSE {"A"}

\* Each record is <<physical value depth, diagnostic path depth>>.
\* A repeated named type copies the prefix built before its recursive field.
\* The value local is rebound at the new depth, but its context is stale.
RECURSIVE Legacy(_, _, _, _)
Legacy(t, active, firstDepth, depth) ==
  IF t \in active
  THEN IF fieldFirst[t] THEN {<<depth, firstDepth[t]>>} ELSE {}
  ELSE {<<depth, depth>>} \cup
       Legacy(NextType(t), active \cup {t},
              [firstDepth EXCEPT ![t] = depth], depth + 1)

Init == /\ mutual \in BOOLEAN
        /\ fieldFirst \in [Types -> BOOLEAN]
        /\ pending = {"A"}
        /\ emitted = {}
        /\ phase = "compile"
Compile == /\ phase = "compile" /\ pending # {}
           /\ \E t \in pending:
                /\ emitted' = emitted \cup {t}
                /\ pending' = (pending \ {t}) \cup ({NextType(t)} \ emitted')
           /\ UNCHANGED <<mutual, fieldFirst, phase>>
Finish == /\ phase = "compile" /\ pending = {}
          /\ phase' = "done"
          /\ UNCHANGED <<mutual, fieldFirst, pending, emitted>>
Next == Compile \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

\* Interpret one complete predicate plus one named edge per declaration.
\* Finite input depth decreases independently of the recursive schema graph.
RECURSIVE Evaluate(_, _, _, _)
Evaluate(t, depth, remaining, path) ==
  IF t \notin emitted THEN {}
  ELSE {<<depth, path>>} \cup
       (IF remaining = 0 THEN {}
        ELSE Evaluate(NextType(t), depth + 1, remaining - 1,
                      IF Mode = "named-static" THEN 0 ELSE path + 1))

\* The root body starts from its object attribute, not a memoized named edge.
Records == IF Mode = "legacy"
           THEN {<<0, 0>>} \cup Legacy(NextType("A"), {}, [t \in Types |-> 0], 1)
           ELSE Evaluate("A", 0, 5, 0)
Checks == {r[1] : r \in Records}
Coverage == phase = "done" => (0..5) \subseteq Checks
Paths == phase = "done" => \A r \in Records: r[1] = r[2]
FiniteDeclarations == /\ emitted \subseteq Reachable
                      /\ pending \subseteq Reachable
                      /\ pending \cap emitted = {}
                      /\ Cardinality(emitted) <= 2
CompleteDeclarations == phase = "done" => emitted = Reachable
Terminates == <> (phase = "done")
=============================================================================
