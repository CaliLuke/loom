--------------------- MODULE InterceptorContract ---------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANTS Repair, Cases
VARIABLES specimen, phase, admitted, signature, bindings, operations,
          receiveBoundary, errorDelegated, context
vars == <<specimen, phase, admitted, signature, bindings, operations,
          receiveBoundary, errorDelegated, context>>
StorageCases == {"normal", "required", "renamed", "incompatible"}
Types(s) == IF s = "incompatible" THEN <<"string", "int">>
            ELSE <<"string", "string">>
Layouts(s) == <<<<"Value", "pointer">>,
                IF s = "required" THEN <<"Value", "value">>
                ELSE IF s = "renamed" THEN <<"StoredValue", "pointer">>
                ELSE <<"Value", "pointer">>>>
Compatible(s) == Types(s)[1] = Types(s)[2]
Generic == {<<"Send", "value">>, <<"SendWithContext", "ctx-value">>,
            <<"Recv", "none">>, <<"RecvWithContext", "ctx">>, <<"Close", "none">>}
RequiredOps(s) == CASE s = "rpc-sse" ->
                  {<<"Send", "ctx-event">>, <<"SendAndClose", "ctx-event">>, <<"SendError", "ctx-id-error">>}
                [] s = "rpc-bidi" ->
                  {<<"SendNotification", "ctx-value">>, <<"SendResponse", "ctx-value">>, <<"SendError", "ctx-error">>, <<"Close", "none">>}
                [] s = "rpc-request" -> {}
                [] OTHER -> Generic
Boundary(s) == IF s \in {"rpc-bidi", "rpc-request"} THEN "endpoint-message"
               ELSE "stream-operation"
Init == /\ specimen \in Cases /\ phase = "analyze"
        /\ admitted = FALSE /\ signature = "unset" /\ bindings = <<>>
        /\ operations = {} /\ receiveBoundary = "unset"
        /\ errorDelegated = TRUE /\ context = "original"
Analyze == /\ phase = "analyze"
           /\ admitted' = IF Repair = "reject-variants" THEN specimen = "normal"
                          ELSE IF Repair \in {"owned", "signature-only"}
                          THEN Compatible(specimen) ELSE TRUE
           /\ signature' = Types(specimen)[1]
           /\ bindings' = IF Repair \in {"owned", "layout-only"}
                           THEN Layouts(specimen) ELSE <<Layouts(specimen)[1], Layouts(specimen)[1]>>
           /\ operations' = IF Repair \in {"owned", "methods-only"}
                             THEN RequiredOps(specimen) ELSE Generic
           /\ receiveBoundary' = IF Repair = "owned" THEN Boundary(specimen) ELSE "stream-operation"
           /\ phase' = "published"
           /\ UNCHANGED <<specimen, errorDelegated, context>>
Next == Analyze \/ (phase = "published" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
LegalAcceptance == phase = "published" => admitted = Compatible(specimen)
SharedSignature == phase = "published" /\ admitted =>
                   \A method \in 1..2: Types(specimen)[method] = signature
LocalStorage == phase = "published" /\ admitted /\ specimen \in StorageCases =>
                bindings = Layouts(specimen)
AdapterImplemented == phase = "published" /\ admitted => operations = RequiredOps(specimen)
MessageBoundary == phase = "published" /\ admitted => receiveBoundary = Boundary(specimen)
ProtocolPreserved == errorDelegated /\ context = "original"
=============================================================================
