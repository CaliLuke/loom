-------------------------- MODULE OperationLease --------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES active, nativeDone, contextDone, callbackRun, success, canceled, returned
vars == <<active, nativeDone, contextDone, callbackRun, success, canceled, returned>>
Init == /\ active = TRUE /\ nativeDone = FALSE /\ contextDone = FALSE
        /\ callbackRun = FALSE /\ success = FALSE /\ canceled = FALSE
        /\ returned = FALSE
CompleteNative == /\ ~nativeDone
                  /\ nativeDone' = TRUE
                  /\ UNCHANGED <<active, contextDone, callbackRun, success, canceled, returned>>
CompleteCall == /\ nativeDone /\ ~returned
                /\ returned' = TRUE
                /\ active' = FALSE
                /\ success' = ~canceled
                /\ UNCHANGED <<nativeDone, contextDone, callbackRun, canceled>>
CancelContext == /\ ~contextDone /\ contextDone' = TRUE
                 /\ UNCHANGED <<active, nativeDone, callbackRun, success, canceled, returned>>
RunCallback == /\ contextDone /\ ~callbackRun
               /\ callbackRun' = TRUE
               /\ canceled' = (canceled \/ active \/ Mode = "unjoined")
               /\ UNCHANGED <<active, nativeDone, contextDone, success, returned>>
Next == CompleteNative \/ CompleteCall \/ CancelContext \/ RunCallback
Spec == Init /\ [][Next]_vars
NoLateAbort == success => ~canceled
=============================================================================
