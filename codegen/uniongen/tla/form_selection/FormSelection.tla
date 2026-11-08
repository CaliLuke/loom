---------------------------- MODULE FormSelection ----------------------------
EXTENDS Naturals
CONSTANT SelectedPresence
VARIABLES tag, input, minimum, phase, accepted, decoded
vars == <<tag, input, minimum, phase, accepted, decoded>>
Values == {"nil", "empty", "nonempty"}
Size(v) == IF v = "nonempty" THEN 1 ELSE 0
Init == /\ tag \in BOOLEAN /\ input \in Values /\ minimum \in 0..1
        /\ phase = "ready" /\ accepted = FALSE /\ decoded = "absent"
Decode == /\ phase = "ready" /\ phase' = "done"
          /\ accepted' = (tag /\ (SelectedPresence \/ Size(input) > 0) /\ Size(input) >= minimum)
          /\ decoded' = IF tag /\ (SelectedPresence \/ Size(input) > 0)
                         THEN IF Size(input) = 0 THEN "empty" ELSE "nonempty"
                         ELSE "absent"
          /\ UNCHANGED <<tag, input, minimum>>
Next == Decode
Expected == tag /\ Size(input) >= minimum
Agreement == phase = "done" => accepted = Expected
NativeEmpty == phase = "done" /\ accepted /\ Size(input) = 0 => decoded = "empty"
Spec == Init /\ [][Next]_vars
=============================================================================
