---------------------- MODULE FinalResultInterceptor ----------------------
EXTENDS Naturals
CONSTANTS MessageBoundary, ForwardView, PayloadAccess, StreamAccess
VARIABLES phase, calls, observed, wireValue, selectedView, wireView, outcome,
          callbackBeforeSend, endpointResult
vars == <<phase, calls, observed, wireValue, selectedView, wireView, outcome,
          callbackBeforeSend, endpointResult>>
Init == /\ phase = "service"
        /\ calls = 0 /\ observed = "none" /\ wireValue = "none"
        /\ selectedView \in {"default", "tiny"} /\ wireView = "none"
        /\ outcome \in {"success", "service-error", "interceptor-error"}
        /\ callbackBeforeSend = FALSE /\ endpointResult = "none"
Service == /\ phase = "service"
           /\ phase' = IF outcome = "service-error" THEN "return" ELSE "message"
           /\ UNCHANGED <<calls, observed, wireValue, selectedView, wireView,
                          outcome, callbackBeforeSend, endpointResult>>
Message == /\ phase = "message"
           /\ calls' = IF MessageBoundary THEN 1 ELSE 0
           /\ observed' = IF MessageBoundary THEN "canonical" ELSE "none"
           /\ callbackBeforeSend' = MessageBoundary
           /\ phase' = IF MessageBoundary /\ outcome = "interceptor-error"
                        THEN "return" ELSE "send"
           /\ UNCHANGED <<wireValue, selectedView, wireView, outcome, endpointResult>>
Send == /\ phase = "send"
        /\ wireValue' = IF MessageBoundary THEN "modified" ELSE "original"
        /\ wireView' = IF ForwardView THEN selectedView ELSE "default"
        /\ phase' = "return"
        /\ UNCHANGED <<calls, observed, selectedView, outcome,
                       callbackBeforeSend, endpointResult>>
Return == /\ phase = "return"
          /\ endpointResult' = "nil"
          /\ calls' = IF ~MessageBoundary /\ (~StreamAccess \/ PayloadAccess) THEN 1 ELSE calls
          /\ observed' = IF ~MessageBoundary /\ (~StreamAccess \/ PayloadAccess) THEN "nil" ELSE observed
          /\ phase' = "done"
          /\ UNCHANGED <<wireValue, selectedView, wireView, outcome, callbackBeforeSend>>
Next == Service \/ Message \/ Send \/ Return
Spec == Init /\ [][Next]_vars
CanonicalResultAccess == calls > 0 /\ outcome # "service-error" => observed = "canonical"
InterceptionBeforeEmission == wireValue # "none" => callbackBeforeSend
ResultMutationReachesWire == wireValue # "none" => wireValue = "modified"
ViewPreserved == wireValue # "none" => wireView = selectedView
ServiceErrorsDoNotEmit == phase = "done" /\ outcome = "service-error" => wireValue = "none"
CallbackErrorStopsEmission == phase = "done" /\ outcome = "interceptor-error" => wireValue = "none"
NoInventedEndpointResult == phase = "done" => endpointResult = "nil"
ExactlyOneResultCallback == phase = "done" /\ outcome # "service-error" => calls = 1
=============================================================================
