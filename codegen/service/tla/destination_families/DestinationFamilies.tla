------------------------ MODULE DestinationFamilies -------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS OwnedDestination, OwnedEmission
Types == {"Choice", "ChoiceKind"}
Services == {"Alpha", "Beta"}
Family(t, n) == IF t = "Choice" THEN {n, n \o "Kind"} ELSE {n}
Candidates(n) == <<n, n \o "2", n \o "3">>
Destinations == {"left", "right"}
VARIABLES pending, used, names, references, remaining, visited, emitted
vars == <<pending, used, names, references, remaining, visited, emitted>>
Init == /\ remaining = Destinations
        /\ visited = {}
        /\ emitted = {}
        /\ pending = Services \X Types
        /\ used = {}
        /\ names = [t \in Types |-> ""]
        /\ references = [s \in Services |-> [t \in Types |-> ""]]
Reference(s,t) ==
 LET available == {i \in 1..3 : Family(t,Candidates(t)[i]) \cap used = {}}
     first == CHOOSE i \in available : \A j \in available : i <= j
     chosen == IF names[t] # "" THEN names[t] ELSE Candidates(t)[first]
     actual == IF OwnedDestination THEN chosen ELSE t
 IN /\ <<s,t>> \in pending
    /\ pending' = pending \ {<<s,t>>}
    /\ used' = used \cup (IF OwnedDestination THEN Family(t,actual) ELSE {actual})
    /\ names' = [names EXCEPT ![t] = actual]
    /\ references' = [references EXCEPT ![s][t] = actual]
    /\ UNCHANGED <<remaining, visited, emitted>>
Emit(p) ==
 LET key == IF OwnedEmission THEN p ELSE "shared-hash"
 IN /\ p \in remaining
    /\ remaining' = remaining \ {p}
    /\ visited' = visited \cup {key}
    /\ emitted' = IF key \in visited THEN emitted ELSE emitted \cup {p}
    /\ UNCHANGED <<pending, used, names, references>>
Next == (\E s \in Services, t \in Types : Reference(s,t))
        \/ (\E p \in Destinations : Emit(p))
        \/ (pending = {} /\ remaining = {} /\ UNCHANGED vars)
EmittedDestinationsComplete == remaining = {} => emitted = Destinations
DeclarationsDistinct == \A a,b \in Types : a # b /\ names[a] # "" /\ names[b] # "" => Family(a,names[a]) \cap Family(b,names[b]) = {}
ReferencesAgree == \A s \in Services, t \in Types : references[s][t] # "" => references[s][t] = names[t]
=============================================================================
