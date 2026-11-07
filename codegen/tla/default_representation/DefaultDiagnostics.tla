------------------------ MODULE DefaultDiagnostics ------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES sample, invoked, outcome, phase
vars == <<sample, invoked, outcome, phase>>
Cases == {
    [gate |-> TRUE, supported |-> TRUE, panic |-> "none"],
    [gate |-> TRUE, supported |-> FALSE, panic |-> "none"],
    [gate |-> FALSE, supported |-> FALSE, panic |-> "none"],
    [gate |-> TRUE, supported |-> TRUE, panic |-> "programmer"],
    [gate |-> TRUE, supported |-> TRUE, panic |-> "attributed-programmer"]
}
Init == /\ sample \in Cases /\ invoked = FALSE /\ outcome = "none" /\ phase = "ready"
Analyze == /\ phase = "ready"
           /\ invoked' = (sample.gate \/ Mode = "preflight")
           /\ outcome' = IF sample.panic # "none"
                THEN IF Mode = "catch-all" THEN "diagnostic" ELSE "panic"
                ELSE IF (sample.gate \/ Mode = "preflight") /\ ~sample.supported
                     THEN IF Mode = "current" THEN "panic" ELSE "diagnostic"
                     ELSE "success"
           /\ phase' = "done" /\ UNCHANGED sample
Next == Analyze
Spec == Init /\ [][Next]_vars
ActualInvocation == phase = "done" => invoked = sample.gate
ExpectedDiagnostic == phase = "done" /\ sample.gate /\ ~sample.supported /\ sample.panic = "none" => outcome = "diagnostic"
ProgrammerPanic == phase = "done" /\ sample.panic # "none" => outcome = "panic"
UnusedDefault == phase = "done" /\ ~sample.gate /\ sample.panic = "none" => outcome = "success"
=============================================================================
