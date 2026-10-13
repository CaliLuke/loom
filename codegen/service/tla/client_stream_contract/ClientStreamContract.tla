------------------------ MODULE ClientStreamContract ------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANTS Repair, Transports
VARIABLES phase, transport, direction, mixed, api, concrete, wrapper, context
vars == <<phase, transport, direction, mixed, api, concrete, wrapper, context>>
SSE(t) == t \in {"http-sse", "rpc-sse"}
Special(t) == SSE(t) \/ t = "rpc-ws"
Close == <<"Close", "none", "error">>
Result == IF mixed THEN "stream-type" ELSE "result-type"
RecvName == IF direction = "upload" THEN "CloseAndRecv" ELSE "Recv"
Send == {<<"Send", "payload", "error">>, <<"SendWithContext", "ctx-payload", "error">>}
Recv == {<<RecvName, "none", Result>>, <<(IF direction = "upload" THEN "CloseAndRecvWithContext" ELSE "RecvWithContext"), "ctx", Result>>}
HasSend == direction \in {"upload", "send-only", "duplex"}
HasRecv == direction # "send-only"
BaseClose == direction \in {"send-only", "duplex"}
Generic == (IF HasSend THEN Send ELSE {}) \cup
           (IF HasRecv THEN Recv ELSE {})
           \cup (IF BaseClose THEN {Close} ELSE {})
Owned == Generic \cup (IF Special(transport) THEN {Close} ELSE {})
ServerOps == {<<"SendNotification", "ctx-payload", "error">>,
              <<"SendResponse", "ctx-payload", "error">>,
              <<"SendError", "ctx-error", "error">>}
OldAPI == IF transport = "rpc-ws"
          THEN (IF HasSend THEN ServerOps ELSE {}) \cup {Close}
          ELSE Generic
OldConcrete == IF SSE(transport)
               THEN {<<"Recv", "ctx", Result>>, Close}
               ELSE Generic \cup (IF transport = "rpc-ws" THEN {Close} ELSE {})
Init == /\ phase = "select"
        /\ transport = "unset" /\ direction = "unset" /\ mixed = FALSE
        /\ api = {} /\ concrete = {} /\ wrapper = {} /\ context = "unset"
Select == /\ phase = "select"
          /\ \E t \in Transports, d \in {"download", "upload", "send-only", "duplex"}, m \in BOOLEAN:
               /\ (~SSE(t) \/ d = "download")
               /\ transport' = t /\ direction' = d /\ mixed' = m
          /\ phase' = "emit"
          /\ UNCHANGED <<api, concrete, wrapper, context>>
Emit == /\ phase = "emit"
        /\ api' = IF Repair = "current" THEN OldAPI
                   ELSE IF Repair = "signatures" THEN Generic \cup (IF transport = "rpc-ws" THEN {Close} ELSE {})
                   ELSE Owned
        /\ concrete' = IF Repair = "current" THEN OldConcrete ELSE Owned
        /\ wrapper' = IF Repair \in {"current", "signatures"} THEN Generic ELSE Owned
        /\ context' = IF Repair = "background" /\ SSE(transport) THEN "background" ELSE "endpoint"
        /\ phase' = "done"
        /\ UNCHANGED <<transport, direction, mixed>>
Next == Select \/ Emit
ServiceImplemented == phase # "done" \/ api \subseteq concrete
WrapperImplemented == phase # "done" \/ api \subseteq wrapper
CleanupReachable == phase # "done" \/ ~Special(transport) \/ Close \in (api \cap wrapper)
OriginalContext == phase # "done" \/ context = "endpoint"
ClientRole == phase # "done" \/ api \cap ServerOps = {}
=============================================================================
