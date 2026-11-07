---------------- MODULE SourceType ----------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES domain, source, nullBranches
vars == <<domain, source, nullBranches>>
Init == /\ domain \in {{"string"}, {"number"}, {"string", "number"}, {}}
        /\ source \in {"string", "number", "absent"}
        /\ nullBranches \in 0..2
Next == UNCHANGED vars
ValidSource == source = "absent" \/ domain \subseteq {source}
Accepted == Policy = "unchecked" \/ ValidSource
OriginalNull == nullBranches = 1
ExportNonNull == IF source = "absent" THEN domain ELSE domain \cap {source}
ExportNull == IF Policy = "checked" \/ source = "absent" THEN OriginalNull ELSE FALSE
DomainPreserved == Accepted => ExportNonNull = domain
NullMultiplicityPreserved == Accepted => ExportNull = OriginalNull
Spec == Init /\ [][Next]_vars
===================================================
