---------------------- MODULE AuthenticationScope ----------------------
EXTENDS Integers, Sequences, FiniteSets
CONSTANT Mode
VARIABLES declarations, noSecurity, result, accepted, phase
vars == <<declarations, noSecurity, result, accepted, phase>>
Kinds == SUBSET {"basic", "session", "anonymous"}
Expand(decl) == (decl \ {"session"}) \cup
                (IF "session" \in decl THEN {"jwt", "cookie"} ELSE {})
DeclaredScopes == {i \in 1..3: declarations[i] # {}}
Nearest == IF DeclaredScopes = {} THEN 1
           ELSE CHOOSE i \in DeclaredScopes: \A j \in DeclaredScopes: j <= i
Candidate == LET methodPair == declarations[3]
                 servicePair == IF methodPair = {} THEN declarations[2] ELSE methodPair
                 apiPair == IF servicePair = {} THEN declarations[1] ELSE servicePair
             IN Expand(apiPair)
Expected == IF noSecurity THEN {} ELSE Expand(declarations[Nearest])
Contradiction == noSecurity /\ declarations[3] \cap {"basic", "session"} # {}
OrdinaryAt(i) == declarations[i] \ {"session"}
OldOrdinary == IF OrdinaryAt(3) # {} THEN OrdinaryAt(3)
               ELSE IF OrdinaryAt(2) # {} THEN OrdinaryAt(2) ELSE OrdinaryAt(1)
OldSession == IF \E i \in 1..3: "session" \in declarations[i]
              THEN {"jwt", "cookie"} ELSE {}
Init == /\ declarations \in [1..3 -> Kinds]
        /\ noSecurity \in BOOLEAN
        /\ result = {} /\ accepted = FALSE /\ phase = "select"
Resolve == /\ phase = "select"
           /\ accepted' = IF Mode = "legacy-admission" THEN TRUE ELSE ~Contradiction
           /\ result' = IF Mode = "legacy-scope" /\ ~noSecurity
                         THEN OldOrdinary \cup OldSession
                         ELSE IF noSecurity THEN {} ELSE Candidate
           /\ phase' = "done"
           /\ UNCHANGED <<declarations, noSecurity>>
Next == Resolve
Spec == Init /\ [][Next]_vars
NearestScope == phase = "done" /\ accepted => result = Expected
ConflictRejected == phase = "done" => accepted = ~Contradiction
=============================================================================
