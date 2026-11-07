----------------------- MODULE CanonicalAcquisition -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode, Equivalent
VARIABLES phase, claimed, originalComplete, visits, accepted, rejected
vars == <<phase, claimed, originalComplete, visits, accepted, rejected>>
Init == /\ phase = "Acquire"
        /\ claimed = FALSE
        /\ originalComplete = FALSE
        /\ visits = 0
        /\ accepted = FALSE
        /\ rejected = FALSE
Acquire == /\ phase = "Acquire"
           /\ claimed' = (Mode # "Unreserved")
           /\ phase' = "Child"
           /\ UNCHANGED <<originalComplete, visits, accepted, rejected>>
Child == /\ phase = "Child"
         /\ visits' = visits + 1
         /\ phase' = IF claimed THEN "CandidateComplete" ELSE "Acquire"
         /\ UNCHANGED <<claimed, originalComplete, accepted, rejected>>
CandidateComplete == /\ phase = "CandidateComplete"
                     /\ phase' = IF Mode = "Early" THEN "Validate" ELSE "OriginalComplete"
                     /\ UNCHANGED <<claimed, originalComplete, visits, accepted, rejected>>
OriginalComplete == /\ phase = "OriginalComplete"
                    /\ originalComplete' = TRUE
                    /\ phase' = "Validate"
                    /\ UNCHANGED <<claimed, visits, accepted, rejected>>
Validate == /\ phase = "Validate"
            /\ accepted' = Equivalent
            /\ rejected' = ~Equivalent
            /\ phase' = "Done"
            /\ UNCHANGED <<claimed, originalComplete, visits>>
Done == /\ phase = "Done" /\ UNCHANGED vars
Next == Acquire \/ Child \/ CandidateComplete \/ OriginalComplete \/ Validate \/ Done
BoundedAcquisition == visits <= 1
CompletedComparison == (accepted \/ rejected) => originalComplete
SoundDecision == phase = "Done" => (accepted = Equivalent /\ rejected = ~Equivalent)
Complete == <>(phase = "Done")
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
=============================================================================
