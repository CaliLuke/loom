-------------------- MODULE DefaultPropagation --------------------
EXTENDS Naturals, TLC
CONSTANT Strategy
VARIABLES supplied, crossing, parserOK, phase, level, error, assigned, published, panicked
vars == <<supplied, crossing, parserOK, phase, level, error, assigned, published, panicked>>

Init == /\ supplied \in BOOLEAN
        /\ crossing \in BOOLEAN
        /\ parserOK \in BOOLEAN
        /\ phase = "leaf"
        /\ level = 0
        /\ error = FALSE
        /\ assigned = "none"
        /\ published = FALSE
        /\ panicked = FALSE

Leaf == /\ phase = "leaf"
        /\ phase' = "propagate"
        /\ level' = 0
        /\ IF supplied /\ ~crossing
              THEN /\ assigned' = "supplied"
                   /\ error' = FALSE
                   /\ panicked' = FALSE
              ELSE IF Strategy = "Literal"
                      THEN /\ assigned' = "native-string"
                           /\ error' = FALSE
                           /\ panicked' = FALSE
                      ELSE IF parserOK
                              THEN /\ assigned' = IF supplied THEN "custom-value" ELSE "custom-default"
                                   /\ error' = FALSE
                                   /\ panicked' = FALSE
                              ELSE /\ assigned' = "none"
                                   /\ error' = TRUE
                                   /\ panicked' = (Strategy = "Panic")
        /\ UNCHANGED <<supplied, crossing, parserOK, published>>

Propagate == /\ phase = "propagate"
             /\ level < 2
             /\ level' = level + 1
             /\ error' = IF Strategy = "Drop" THEN FALSE ELSE error
             /\ UNCHANGED <<supplied, crossing, parserOK, phase, assigned, published, panicked>>

Finish == /\ phase = "propagate"
          /\ level = 2
          /\ phase' = "done"
          /\ published' = (~error /\ ~panicked)
          /\ UNCHANGED <<supplied, crossing, parserOK, level, error, assigned, panicked>>

Next == Leaf \/ Propagate \/ Finish
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
TypedAssignment == assigned # "native-string"
NoPanic == ~panicked
FailureVisible == phase = "done" /\ (~supplied \/ crossing) /\ ~parserOK => error /\ ~published
PreserveSupplied == supplied /\ ~crossing /\ phase = "done" => assigned = "supplied" /\ published
ApplyDefault == ~supplied /\ parserOK /\ phase = "done" => assigned = "custom-default" /\ published
ConvertSupplied == supplied /\ crossing /\ parserOK /\ phase = "done" => assigned = "custom-value" /\ published
Completion == <> (phase = "done")
====================================================================
