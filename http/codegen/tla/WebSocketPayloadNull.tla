---------------------- MODULE WebSocketPayloadNull ----------------------
EXTENDS TLC
CONSTANT RejectNullablePayload
VARIABLES phase, transport, direction, nullable, shape, message, wire, received
vars == <<phase, transport, direction, nullable, shape, message, wire, received>>

Init == /\ phase = "validate"
        /\ transport \in {"HTTP", "JSONRPC"}
        /\ direction \in {"request", "response"}
        /\ nullable \in BOOLEAN
        /\ shape \in {"scalar", "object", "array", "map"}
        /\ message \in IF nullable
                      THEN {"value", "null", "nested-null", "end"}
                      ELSE {"value", "nested-null", "end"}
        /\ wire = "none" /\ received = "none"

UsesNullSentinel == transport = "HTTP" /\ direction = "request"
Forbidden == RejectNullablePayload /\ UsesNullSentinel /\ nullable

Validate == /\ phase = "validate"
            /\ phase' = IF Forbidden THEN "rejected" ELSE "encode"
            /\ UNCHANGED <<transport, direction, nullable, shape, message, wire, received>>

Encode == /\ phase = "encode"
          /\ wire' = IF message = "end"
                     THEN IF UsesNullSentinel THEN "null" ELSE "close"
                     ELSE message
          /\ phase' = "decode"
          /\ UNCHANGED <<transport, direction, nullable, shape, message, received>>

Decode == /\ phase = "decode"
          /\ received' = IF wire = "close" \/ (UsesNullSentinel /\ wire = "null")
                         THEN "end" ELSE wire
          /\ phase' = "done"
          /\ UNCHANGED <<transport, direction, nullable, shape, message, wire>>

Next == Validate \/ Encode \/ Decode
Spec == Init /\ [][Next]_vars
AcceptedMessagesPreserved == phase = "done" => received = message
SupportedContractsAccepted == phase = "rejected" => UsesNullSentinel /\ nullable
=============================================================================
