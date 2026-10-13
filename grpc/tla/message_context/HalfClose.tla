-------------------------- MODULE HalfClose --------------------------
EXTENDS Naturals
CONSTANT Strategy
VARIABLES terminal, released, closeState, result, nativeResult, admitted,
          calls, requests, lateIO
vars == <<terminal, released, closeState, result, nativeResult, admitted,
          calls, requests, lateIO>>
Init == /\ terminal = FALSE /\ released = FALSE
        /\ closeState = "idle" /\ result = "none"
        /\ nativeResult = "none" /\ admitted = FALSE
        /\ calls = 0 /\ requests = 0 /\ lateIO = FALSE
Terminal == /\ ~terminal
            /\ terminal' = TRUE
            /\ released' = (Strategy # "retain")
            /\ UNCHANGED <<closeState, result, nativeResult, admitted, calls, requests, lateIO>>
Close == /\ closeState = "idle" /\ requests < 2
         /\ requests' = requests + 1
         /\ IF terminal /\ Strategy # "retain"
               THEN /\ closeState' = "done"
                    /\ result' = IF Strategy = "classified" THEN "nil" ELSE "canceled"
                    /\ admitted' = FALSE
                    /\ UNCHANGED <<calls, lateIO>>
               ELSE /\ closeState' = "active"
                    /\ result' = "none"
                    /\ admitted' = TRUE
                    /\ calls' = calls + 1
                    /\ lateIO' = lateIO \/ terminal
         /\ UNCHANGED <<terminal, released, nativeResult>>
Native == /\ closeState = "active" /\ nativeResult = "none" /\ ~released
          /\ nativeResult' \in {"nil", "native-error"}
          /\ UNCHANGED <<terminal, released, closeState, result, admitted, calls, requests, lateIO>>
Return == /\ closeState = "active"
          /\ (nativeResult # "none" \/ released)
          /\ closeState' = "done"
          /\ result' = IF nativeResult # "none" THEN nativeResult
                        ELSE IF Strategy = "classified" THEN "nil" ELSE "canceled"
          /\ UNCHANGED <<terminal, released, nativeResult, admitted, calls, requests, lateIO>>
Again == /\ closeState = "done" /\ requests < 2
         /\ closeState' = "idle" /\ nativeResult' = "none"
         /\ result' = "none"
         /\ UNCHANGED <<terminal, released, admitted, calls, requests, lateIO>>
Next == Terminal \/ Close \/ Native \/ Return \/ Again
Spec == Init /\ [][Next]_vars
ReleaseTerminal == terminal => released
NoLateNativeAdmission == ~lateIO
NoInternalCancellation == closeState = "done" => result # "canceled"
PreserveNativeResult == closeState = "done" /\ admitted /\ nativeResult # "none" => result = nativeResult
=============================================================================
