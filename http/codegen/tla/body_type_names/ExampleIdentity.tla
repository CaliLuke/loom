------------------------ MODULE ExampleIdentity ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Types == {"body", "authored"}
Key(t) == <<IF Mode = "checked" THEN t ELSE "shared", "svc#OtherRequestBody">>
VARIABLES remaining, cached, values, results
vars == <<remaining, cached, values, results>>
Init == /\ remaining = Types
        /\ cached = {}
        /\ values = [k \in {Key(t): t \in Types} |-> ""]
        /\ results = [t \in Types |-> ""]
Generate(t) == /\ t \in remaining
               /\ results' = [results EXCEPT ![t] = IF Key(t) \in cached THEN values[Key(t)] ELSE t]
               /\ values' = IF Key(t) \in cached THEN values ELSE [values EXCEPT ![Key(t)] = t]
               /\ cached' = cached \cup {Key(t)}
               /\ remaining' = remaining \ {t}
Next == (\E t \in remaining: Generate(t)) \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ExamplesPreserved == \A t \in Types \ remaining: results[t] = t
Terminates == <>(remaining = {})
=============================================================================
