---------------- MODULE OccurrenceOwnership ----------------
EXTENDS Naturals
CONSTANT Policy
VARIABLE named, nullable, annotated, phase, declaration, outer, inner, restrictsNull, acceptsNull
vars == <<named, nullable, annotated, phase, declaration, outer, inner, restrictsNull, acceptsNull>>
Init == /\ named \in BOOLEAN
        /\ nullable \in BOOLEAN
        /\ annotated \in BOOLEAN
        /\ restrictsNull \in BOOLEAN
        /\ acceptsNull = nullable
        /\ phase = "source"
        /\ declaration = "declared"
        /\ outer = annotated /\ inner = FALSE
SourceWrapper == /\ phase = "source"
                 /\ declaration' = IF Policy = "old" /\ named
                                    THEN "occurrence-source" ELSE declaration
                 /\ outer' = IF Policy = "old" THEN FALSE ELSE outer
                 /\ inner' = IF Policy = "old" THEN outer ELSE FALSE
                 /\ acceptsNull' = (acceptsNull /\ ~restrictsNull)
                 /\ phase' = IF nullable /\ (Policy # "unwrapped" \/ ~named) THEN "nullable" ELSE "done"
                 /\ UNCHANGED <<named, nullable, annotated, restrictsNull>>
\* Nullable retains only annotations on the schema it receives; it cannot
\* recover annotations incorrectly pushed into a source wrapper's child.
NullableWrapper == /\ phase = "nullable" /\ phase' = "done"
                   /\ acceptsNull' = TRUE
                   /\ UNCHANGED <<named, nullable, annotated, declaration, outer, inner, restrictsNull>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == SourceWrapper \/ NullableWrapper \/ Done
CanonicalDeclarationStable == declaration = "declared"
OccurrenceAnnotations == phase = "done" => (outer = annotated /\ ~inner)
NullPreserved == phase = "done" => (acceptsNull = nullable)
Spec == Init /\ [][Next]_vars
=============================================================
