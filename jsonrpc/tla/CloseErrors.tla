-------------------------- MODULE CloseErrors --------------------------
EXTENDS Naturals
CONSTANT Design
VARIABLES stream, connection, lastRef, pending, operation, phase, cause, matched
vars == <<stream, connection, lastRef, pending, operation, phase, cause, matched>>

Init == /\ stream = "open" /\ connection = "open"
        /\ lastRef \in BOOLEAN /\ pending \in BOOLEAN
        /\ operation = "none" /\ phase = "idle"
        /\ cause = "none" /\ matched = FALSE

\* CancelStream models completion of contextDone, not merely cancel(). The
\* operation has an independent live context. A client close records its
\* reason before closing the socket; an earlier connection failure wins.
CloseStream == /\ stream = "open" /\ stream' = "close"
               /\ connection' = IF lastRef /\ connection = "open"
                                  THEN "close" ELSE connection
               /\ UNCHANGED <<lastRef, pending, operation, phase, cause, matched>>
CancelStream == /\ stream = "open" /\ stream' = "cancel"
                /\ connection' = IF lastRef /\ connection = "open"
                                   THEN "close" ELSE connection
                /\ UNCHANGED <<lastRef, pending, operation, phase, cause, matched>>
EndConnection(reason) == /\ connection = "open" /\ connection' = reason
                         /\ UNCHANGED <<stream, lastRef, pending, operation, phase, cause, matched>>

Reason == IF stream # "open" THEN stream
          ELSE IF connection # "open" THEN connection ELSE "none"

\* Start corresponds to the preflight checks under the stream/connection
\* locks. The as-is empty Recv path never inspects the connection error.
Start(op) ==
    /\ phase = "idle" /\ operation' = op
    /\ IF Reason # "none"
       THEN /\ phase' = "returned" /\ cause' = Reason
            /\ matched' = IF Design = "complete" THEN Reason = "close"
                          ELSE Design = "sentinels" /\
                               (Reason = "close" \/ (stream = "cancel" /\ op # "recv")) /\
                               ~(op = "recv" /\ ~pending /\ stream = "open")
       ELSE /\ cause' = "none" /\ matched' = FALSE
            /\ phase' = IF op = "recv" THEN
                            IF pending THEN "await" ELSE "returned"
                        ELSE "write"
    /\ UNCHANGED <<stream, connection, lastRef, pending>>

\* A Send/Notify/Call can pass preflight, then fail in WriteJSON because the
\* client or the last stream closed the socket. Merely changing the sentinel
\* declarations does not classify that in-flight failure.
Write ==
    /\ phase = "write"
    /\ IF connection # "open"
       THEN /\ phase' = "returned"
            /\ cause' = Reason
            /\ matched' = (Design = "complete" /\ cause' = "close")
       ELSE /\ phase' = IF operation = "call" THEN "await" ELSE "returned"
            /\ cause' = "none" /\ matched' = FALSE
    /\ UNCHANGED <<stream, connection, lastRef, pending, operation>>

\* A response already available may win against closure. This model makes
\* no claim that Close revokes an already received successful response.
Response == /\ phase = "await" /\ phase' = "returned"
            /\ cause' = "none" /\ matched' = FALSE
            /\ UNCHANGED <<stream, connection, lastRef, pending, operation>>
Wake == /\ phase = "await" /\ Reason # "none"
        /\ phase' = "returned" /\ cause' = Reason
        /\ matched' = (Design # "asis" /\ Reason = "close")
        /\ UNCHANGED <<stream, connection, lastRef, pending, operation>>

Next == CloseStream \/ CancelStream \/ EndConnection("close") \/
        EndConnection("failure") \/ Write \/ Response \/ Wake \/
        (\E op \in {"send", "notify", "call", "recv"}: Start(op))
Spec == Init /\ [][Next]_vars
ClosureMatches == phase = "returned" /\ cause = "close" => matched
OtherCausesStayDistinct == phase = "returned" /\ cause # "close" => ~matched
PendingRequest == pending
=============================================================================
