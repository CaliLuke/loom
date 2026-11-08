----------------------------- MODULE ClaimBinding -----------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode, Reverse
VARIABLES requests, cursor, registrations, selected
vars == <<requests,cursor,registrations,selected>>
(* Tokens are already captured prepared target/node identities, not endpoint
   policy enums. qAgain denotes actual reentry through the identical binding. *)
Requests == {"q","f","h","qAgain","body1","body2"}
Binding(r) == CASE r \in {"q","qAgain"} -> "Q"
                  [] r = "f" -> "F"
                  [] r = "h" -> "H"
                  [] r = "body1" -> "B1"
                  [] OTHER -> "B2"
Fingerprint(r) == IF r \in {"body1","body2"} THEN r ELSE "text-F"
Contract(r) == IF r \in {"body1","body2"} THEN "JSON" ELSE "text"
ExpectedOwner(r) == IF r \in {"body1","body2"} THEN "declaration-binding" ELSE Binding(r)
Init == /\ requests = IF Reverse THEN <<"f","h","q","body2","qAgain","body1">>
                          ELSE <<"q","qAgain","body1","f","body2","h">>
        /\ cursor = 1
        /\ registrations = <<[fp |-> "declaration-F", contract |-> "JSON",
                               public |-> TRUE, claim |-> "Name", binding |-> "declaration-binding"]>>
        /\ selected = <<>>
Eligible(r,i) == LET entry == registrations[i] IN
    IF Mode = "Fresh" THEN FALSE
    ELSE IF entry.public THEN (Mode = "BroadPublic" \/ entry.contract = Contract(r))
    ELSE entry.fp = Fingerprint(r) /\ entry.claim = "Name" /\
         (Mode = "BroadFlag" \/ (Mode # "NoBindingReuse" /\ entry.binding = Binding(r)))
Select == /\ cursor <= Len(requests)
          /\ LET r == requests[cursor]
                 eligible == {i \in 1..Len(registrations): Eligible(r,i)}
                 fresh == eligible = {}
                 handle == IF fresh THEN Len(registrations)+1 ELSE CHOOSE i \in eligible: \A j \in eligible: i <= j
             IN /\ registrations' = IF fresh THEN Append(registrations,
                    [fp |-> Fingerprint(r), contract |-> Contract(r), public |-> FALSE,
                     claim |-> "Name", binding |-> Binding(r)]) ELSE registrations
                /\ selected' = Append(selected,handle)
          /\ cursor' = cursor+1
          /\ UNCHANGED requests
Done == /\ cursor > Len(requests) /\ UNCHANGED vars
Next == Select \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
BindingPreserved == \A n \in 1..Len(selected): registrations[selected[n]].binding = ExpectedOwner(requests[n])
SameOwnerReentry == \A i,j \in 1..Len(selected):
    Binding(requests[i]) = Binding(requests[j]) => selected[i] = selected[j]
PublicCanonicalOwner == \A n \in 1..Len(selected): requests[n] \in {"body1","body2"} => selected[n] = 1
CompleteBindings == cursor > Len(requests) =>
    {registrations[i].binding: i \in 1..Len(registrations)} = {"declaration-binding","Q","F","H"}
MinimalReservations == cursor > Len(requests) => Len(registrations) = 4
Terminates == <>(cursor > Len(requests))
===============================================================================
