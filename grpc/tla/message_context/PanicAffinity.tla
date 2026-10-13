-------------------------- MODULE PanicAffinity --------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES terminal, servicePanic, relayed, escaped, contained, done
vars == <<terminal, servicePanic, relayed, escaped, contained, done>>
Init == /\ terminal = FALSE /\ servicePanic = FALSE /\ relayed = FALSE
        /\ escaped = FALSE /\ contained = FALSE /\ done = FALSE
Terminate == /\ ~terminal /\ ~done /\ terminal' = TRUE
             /\ UNCHANGED <<servicePanic, relayed, escaped, contained, done>>
Panic == /\ ~servicePanic /\ servicePanic' = TRUE
         /\ UNCHANGED <<terminal, relayed, escaped, contained, done>>
Handle == /\ servicePanic /\ ~done /\ done' = TRUE
          /\ relayed' = (Mode = "same-stack" \/ (Mode \in {"relay", "native"} /\ ~terminal))
          /\ escaped' = (Mode = "naive-worker")
          /\ contained' = (Mode \in {"relay", "native"} /\ terminal)
          /\ UNCHANGED <<terminal, servicePanic>>
Next == Terminate \/ Panic \/ Handle
Spec == Init /\ [][Next]_vars
NoEscapedWorkerPanic == ~escaped
OuterRecoveryPreserved == done => relayed
ActiveNativeRecovery == done /\ ~terminal => relayed
LateNativeContained == done /\ terminal => contained
=============================================================================
