------------------------ MODULE ExternalTypeNames ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Services == {"first", "second"}
Names == {"Moved", "Moved2"}
VARIABLES original, local, remaining, declarations
vars == <<original, local, remaining, declarations>>
Init == /\ original \in [Services -> SUBSET Names]
        /\ local = original
        /\ remaining = Services
        /\ declarations = [s \in Services |-> ""]
LocalName(s) == IF "Moved" \notin local[s] THEN "Moved"
                ELSE IF "Moved2" \notin local[s] THEN "Moved2" ELSE "Moved3"
Declare(s) == LET name == IF Mode = "legacy" THEN LocalName(s) ELSE "Moved" IN
              /\ s \in remaining
              /\ declarations' = [declarations EXCEPT ![s] = name]
              /\ local' = [local EXCEPT ![s] = @ \cup {LocalName(s)}]
              /\ remaining' = remaining \ {s}
              /\ UNCHANGED original
Next == (\E s \in remaining: Declare(s))
        \/ (remaining = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
ReferenceResolves == \A s \in Services \ remaining: declarations[s] = "Moved"
SharedDeclaration == \A a,b \in Services \ remaining: declarations[a] = declarations[b]
ReservationsPreserved == \A s \in Services: original[s] \subseteq local[s]
DeclarationReserved == \A s \in Services \ remaining: declarations[s] \in local[s]
Terminates == <>(remaining = {})
=============================================================================
