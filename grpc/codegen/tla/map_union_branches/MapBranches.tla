---------------------------- MODULE MapBranches ----------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES chosen, entries, phase, accepted, wireKind, wireEntries, decodedKind, decodedEntries
vars == <<chosen, entries, phase, accepted, wireKind, wireEntries, decodedKind, decodedEntries>>
Kinds == {"unset", "map", "leaf"}
Contents == {"nil", "empty", "populated"}
Normalize(c) == IF c = "nil" THEN "empty" ELSE c
Init == /\ chosen \in Kinds
        /\ entries \in Contents
        /\ phase = "validate"
        /\ accepted = FALSE
        /\ wireKind = "unset" /\ wireEntries = "empty"
        /\ decodedKind = "unset" /\ decodedEntries = "empty"
Validate == /\ phase = "validate"
            /\ accepted' = ~(Mode = "legacy" /\ chosen = "map")
            /\ phase' = "encode"
            /\ UNCHANGED <<chosen, entries, wireKind, wireEntries, decodedKind, decodedEntries>>
Encode == /\ phase = "encode"
          /\ wireKind' = IF Mode = "drop-empty" /\ chosen = "map" /\ entries # "populated" THEN "unset" ELSE chosen
          /\ wireEntries' = IF chosen = "map" THEN Normalize(entries) ELSE "empty"
          /\ phase' = "decode"
          /\ UNCHANGED <<chosen, entries, accepted, decodedKind, decodedEntries>>
Decode == /\ phase = "decode"
          /\ decodedKind' = wireKind
          /\ decodedEntries' = wireEntries
          /\ phase' = "done"
          /\ UNCHANGED <<chosen, entries, accepted, wireKind, wireEntries>>
Next == Validate \/ Encode \/ Decode \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NamedMapSupported == phase # "validate" => accepted
SelectedBranchPreserved == phase = "done" => decodedKind = chosen
EntriesPreserved == phase = "done" /\ chosen = "map" => decodedEntries = Normalize(entries)
Terminates == <>(phase = "done")
=============================================================================
