------------------------ MODULE ProjectNullability ------------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Mode
VARIABLES source, declarations, occurrences, requests, cursor
vars == <<source, declarations, occurrences, requests, cursor>>
Types == {"ordinary", "nullable"}
Views == {"default", "tiny"}
Edges == {"plain", "local"}
Requests == [typ : Types, view : Views, edge : Edges]
Authored == [t \in Types |-> t = "nullable"]
Key(r) == <<r.typ, IF Mode = "view-conflation" THEN "default" ELSE r.view>>
Init == /\ source = Authored
        /\ declarations = [k \in {} |-> FALSE]
        /\ occurrences = <<>>
        /\ requests \in [1..3 -> Requests]
        /\ cursor = 1
Project ==
    /\ cursor <= 3
    /\ LET r == requests[cursor]
           key == Key(r)
           local == r.edge = "local"
           nullable == IF Mode = "current" THEN FALSE
                       ELSE IF Mode = "promote-occurrence" THEN source[r.typ] \/ local
                       ELSE source[r.typ]
           declaration == [nullable |-> nullable, view |-> r.view]
           cache == IF key \in DOMAIN declarations THEN declarations
                    ELSE declarations @@ (key :> declaration)
       IN /\ declarations' = cache
          /\ occurrences' = Append(occurrences,
                 [request |-> r, key |-> key, local |-> local,
                  nullable |-> cache[key].nullable \/ local])
          /\ source' = IF Mode = "mutate-source"
                        THEN [source EXCEPT ![r.typ] = @ \/ local] ELSE source
    /\ cursor' = cursor + 1
    /\ UNCHANGED requests
Spec == Init /\ [][Project]_vars
SourceUnchanged == source = Authored
DeclarationContract == \A key \in DOMAIN declarations:
    declarations[key].nullable = Authored[key[1]]
SelectedView == \A i \in 1..Len(occurrences):
    declarations[occurrences[i].key].view = occurrences[i].request.view
NullAdmission == \A i \in 1..Len(occurrences):
    occurrences[i].nullable =
      (Authored[occurrences[i].request.typ] \/ occurrences[i].local)
OccurrencePreserved == \A i \in 1..Len(occurrences):
    occurrences[i].local = (occurrences[i].request.edge = "local")
ReuseIdentity == \A i, j \in 1..Len(occurrences):
    (occurrences[i].key = occurrences[j].key) =
      (occurrences[i].request.typ = occurrences[j].request.typ /\
       occurrences[i].request.view = occurrences[j].request.view)
=============================================================================
