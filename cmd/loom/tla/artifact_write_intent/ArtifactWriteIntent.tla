------------------------ MODULE ArtifactWriteIntent ------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES pc, live, user, owned, oldGen, reported, checkedAbsent
vars == <<pc, live, user, owned, oldGen, reported, checkedAbsent>>
Init == /\ pc = "stage" /\ live \in {"absent", "user"}
        /\ user = (live = "user") /\ owned = FALSE /\ oldGen = TRUE
        /\ reported = FALSE /\ checkedAbsent = FALSE
Stage == /\ pc = "stage" /\ pc' = "check"
         /\ UNCHANGED <<live, user, owned, oldGen, reported, checkedAbsent>>
Check == /\ pc = "check" /\ pc' = "install"
         /\ checkedAbsent' = (live = "absent")
         /\ UNCHANGED <<live, user, owned, oldGen, reported>>
UserCreate == /\ pc \in {"stage", "check", "install"} /\ live = "absent"
              /\ live' = "user" /\ user' = TRUE
              /\ UNCHANGED <<pc, owned, oldGen, reported, checkedAbsent>>
Install == /\ pc = "install" /\ pc' = "second"
           /\ LET write == CASE Mode = "lost" -> TRUE
                                [] Mode = "checked" -> checkedAbsent
                                [] OTHER -> live = "absent"
              IN /\ live' = IF write THEN "generated" ELSE live
                 /\ owned' = write /\ reported' = write
           /\ oldGen' = FALSE /\ UNCHANGED <<user, checkedAbsent>>
Success == /\ pc = "second" /\ pc' = "done"
           /\ UNCHANGED <<live, user, owned, oldGen, reported, checkedAbsent>>
FailSecond == /\ pc = "second" /\ pc' = "failed"
              /\ live' = IF owned THEN "absent" ELSE live
              /\ owned' = FALSE /\ reported' = FALSE /\ oldGen' = TRUE
              /\ UNCHANGED <<user, checkedAbsent>>
Next == Stage \/ Check \/ UserCreate \/ Install \/ Success \/ FailSecond
Spec == Init /\ [][Next]_vars
PreserveUserContent == user => live = "user"
SkipNeverOwned == live = "user" => ~owned /\ ~reported
Rollback == pc = "failed" => oldGen /\ ~owned /\ ~reported
=============================================================================
