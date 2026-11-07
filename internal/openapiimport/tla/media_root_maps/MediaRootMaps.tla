---------------- MODULE MediaRootMaps ----------------
EXTENDS Naturals
CONSTANT Policy
VARIABLE media, multiple, depth
vars == <<media, multiple, depth>>
Init == /\ media \in {"form", "multipart"}
        /\ multiple \in BOOLEAN
        /\ depth \in 0..2
Next == UNCHANGED vars
Expected == media = "form" \/ (~multiple /\ depth = 0)
OldAdmission == IF depth > 0 THEN media = "form" ELSE ~multiple
Admitted == IF Policy = "old" THEN OldAdmission ELSE Expected
AdmissionAgreement == Admitted = Expected
FormComposition == media = "form" => Admitted
MultipartCustomBoundary == media = "multipart" => (Admitted = (~multiple /\ depth = 0))
Spec == Init /\ [][Next]_vars
======================================================
