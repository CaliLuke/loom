-------------------- MODULE ConstraintFactsSplit --------------------
EXTENDS TLC
CONSTANT Policy
VARIABLES stage, localValid, boundsValid, hasDefault, semanticAccepted,
          wrapped, predicateAccepted, generationAccepted, wrapperRequired
vars == <<stage, localValid, boundsValid, hasDefault, semanticAccepted,
          wrapped, predicateAccepted, generationAccepted, wrapperRequired>>
Init == /\ stage = "authored"
        /\ localValid \in BOOLEAN /\ boundsValid \in BOOLEAN /\ hasDefault \in BOOLEAN
        /\ semanticAccepted = FALSE /\ wrapped = FALSE
        /\ predicateAccepted = FALSE /\ generationAccepted = FALSE /\ wrapperRequired = FALSE
Admit == /\ stage = "authored" /\ stage' = "admitted"
         /\ semanticAccepted' = (boundsValid /\ (~hasDefault \/ localValid))
         /\ UNCHANGED <<localValid, boundsValid, hasDefault, wrapped,
                         predicateAccepted, generationAccepted, wrapperRequired>>
Lower == /\ stage = "admitted" /\ stage' = "physical"
         /\ wrapped' = TRUE /\ wrapperRequired' = TRUE
         /\ UNCHANGED <<localValid, boundsValid, hasDefault, semanticAccepted,
                         predicateAccepted, generationAccepted>>
Query == /\ stage = "physical" /\ stage' = "queried"
         /\ predicateAccepted' = boundsValid
         /\ generationAccepted' = CASE Policy = "current" ->
                (semanticAccepted /\ boundsValid /\ (~hasDefault \/ ~wrapped))
              [] Policy = "split" -> (semanticAccepted /\ boundsValid)
         /\ UNCHANGED <<localValid, boundsValid, hasDefault, semanticAccepted,
                         wrapped, wrapperRequired>>
Next == Admit \/ Lower \/ Query
Spec == Init /\ [][Next]_vars
ValidDefaultsSurvive == stage = "queried" /\ semanticAccepted => generationAccepted
AuthoredRejectionPreserved == stage = "queried" /\ ~semanticAccepted => ~generationAccepted
PhysicalBoundsPreserved == stage = "queried" => (predicateAccepted = boundsValid)
WrapperRulesPreserved == stage \in {"physical", "queried"} => wrapperRequired
===================================================================
