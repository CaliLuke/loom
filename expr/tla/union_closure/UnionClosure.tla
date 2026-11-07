----------------------------- MODULE UnionClosure -----------------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
Occurrences == 1..3
Requested(i) == i # 1
Policy(i) == IF Requested(i) THEN "closed" ELSE "open"
Contract(i) == IF Mode \in {"distinct", "nominal"} /\ i = 3 THEN "B" ELSE "A"
Expected(i) == Policy(i) \o Contract(i)
Keys == {"openA", "closedA", "closedB"}
Key(i) == IF Mode = "collapse" THEN "openA"
          ELSE IF Mode = "nominal" THEN Policy(i) \o "A" ELSE Expected(i)
VARIABLES remaining, sourceClosed, reference, declaration, emitted
vars == <<remaining, sourceClosed, reference, declaration, emitted>>
Init == /\ remaining = Occurrences
        /\ sourceClosed = FALSE
        /\ reference = [i \in Occurrences |-> "openA"]
        /\ declaration = [k \in Keys |-> "missing"]
        /\ emitted = [k \in Keys |-> 0]
Bind(i) == /\ i \in remaining
           /\ remaining' = remaining \ {i}
           /\ sourceClosed' = IF Mode = "mutate" THEN sourceClosed \/ Requested(i)
                               ELSE sourceClosed
           /\ reference' = [reference EXCEPT ![i] = Key(i)]
           /\ declaration' = IF declaration[Key(i)] = "missing"
                              THEN [declaration EXCEPT ![Key(i)] = Expected(i)]
                              ELSE declaration
           /\ emitted' = IF declaration[Key(i)] = "missing"
                          THEN [emitted EXCEPT ![Key(i)] = @ + 1]
                          ELSE emitted
Next == (\E i \in remaining: Bind(i))
        \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
SourceUnchanged == sourceClosed = FALSE
ReferencePolicy == \A i \in Occurrences \ remaining:
                     declaration[reference[i]] = Expected(i)
NoDuplicateDeclarations == \A k \in Keys: emitted[k] <= 1
Terminates == <>(remaining = {})
===============================================================================
