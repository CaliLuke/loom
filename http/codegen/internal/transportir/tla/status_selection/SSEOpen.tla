------------------------- MODULE SSEOpen -------------------------
\* #976 SSE success handshake, derived from the section 3 selection mode of
\* the SSE endpoint's success responses. A singleton exact success (200 or
\* 201) publishes its literal status; several successes or any non-exact
\* success use status mode, whose typed handshake metadata must select a key
\* before every open trigger (Open, SendComment, Send, Complete with a
\* message, empty Complete, explicit Close, successful handler finalization). Handshake
\* headers may be required or optional in either mode. Metadata is immutable
\* after open; events are a separate carrier. Errors before open use ordinary
\* error mapping; errors after open end the stream. The client selects the key
\* before constructing the stream and exposes the status with zero events.
EXTENDS Naturals, Sequences
CONSTANTS Policy, MaxSteps
VARIABLES config, hdr, meta, opened, published, events, steps, failed, client
vars == <<config, hdr, meta, opened, published, events, steps, failed, client>>

\* Success key sets: exact 200; exact 201; exact 200 and exact 201; 2XX
\* (body-bearing, so 204 is outside the domain).
Configs == {"exact200", "exact201", "multiExact", "class2XX"}
Domain == CASE config = "exact200" -> {200}
            [] config = "exact201" -> {201}
            [] OTHER -> {200, 201}
\* Section 3 mode: singleton exact is literal; everything else is status mode.
Literal == config \in {"exact200", "exact201"}
LiteralStatus == IF config = "exact201" THEN 201 ELSE 200
\* Logical "end" is successful finalization, never unconditional Go End cleanup.
\* Explicit Close shares the logical "complete"/"end" opening transition;
\* SSETerminal separately models its result and terminal-admission semantics.
Triggers == {"open", "comment", "event", "complete", "end"}
NoStatus == 0
ServiceError == 404

Init == /\ config \in Configs
        /\ hdr \in {"none", "optional", "required"}
        /\ meta = [status |-> NoStatus, header |-> FALSE]
        /\ opened = FALSE /\ published = <<>> /\ events = 0
        /\ steps = 0 /\ failed = FALSE
        /\ client = [status |-> 0, events |-> 0, kind |-> "none"]

\* The status a writer commits at the first open trigger.
OpenStatus(firstEvent) ==
  CASE Policy = "status-200-special" /\ config \in {"exact200", "exact201", "multiExact"} -> 200
    [] Literal -> LiteralStatus
    [] Policy = "first-event-status" -> IF firstEvent THEN 201 ELSE 200
    [] OTHER -> meta.status
HeaderOK == hdr = "required" => (meta.header \/ Policy = "skip-header-check")
Publish(firstEvent) ==
  LET st == OpenStatus(firstEvent) IN
  IF st \in Domain /\ HeaderOK
  THEN /\ published' = Append(published, st) /\ opened' = TRUE /\ failed' = failed
  \* Missing selection, out-of-domain status or a missing required header:
  \* pre-commit framework error through the error path, never a guessed status.
  ELSE /\ published' = Append(published, 500) /\ opened' = FALSE /\ failed' = TRUE

\* SetResponse exists in status mode, and for a singleton exact success only
\* when its handshake declares headers; a literal handshake has no StatusField.
SetResponse == /\ steps < MaxSteps /\ ~failed
               /\ (~Literal \/ hdr # "none")
               /\ \E st \in (IF Literal THEN {NoStatus} ELSE {200, 201, 204}), h \in BOOLEAN :
                    IF opened /\ Policy # "mutable-after-open"
                    THEN UNCHANGED meta
                    ELSE meta' = [status |-> st, header |-> h /\ hdr # "none"]
               /\ steps' = steps + 1
               /\ UNCHANGED <<config, hdr, opened, published, events, failed, client>>
Trigger == /\ steps < MaxSteps /\ ~failed /\ Len(published) <= 1
           /\ \E t \in Triggers :
                /\ IF opened
                   THEN UNCHANGED <<published, opened, failed>>
                   ELSE IF t = "end" /\ Policy = "end-unpublished"
                   THEN UNCHANGED <<published, opened, failed>>
                   ELSE Publish(t = "event")
                /\ events' = IF t = "event" /\ (opened \/ (OpenStatus(TRUE) \in Domain /\ HeaderOK))
                             THEN events + 1 ELSE events
           /\ steps' = steps + 1
           /\ UNCHANGED <<config, hdr, meta, client>>
Fail == /\ steps < MaxSteps /\ ~failed
        /\ IF ~opened \/ Policy = "post-open-remap"
           THEN published' = Append(published, ServiceError)
           ELSE UNCHANGED published
        /\ failed' = TRUE /\ steps' = steps + 1
        /\ UNCHANGED <<config, hdr, meta, opened, events, client>>
\* Handler return with an unopened stream and no error still owes a response.
Return == /\ ~failed /\ published = <<>>
          /\ IF Policy = "end-unpublished"
             THEN published' = <<200>> /\ opened' = TRUE /\ failed' = FALSE
             ELSE Publish(FALSE)
          /\ steps' = MaxSteps
          /\ UNCHANGED <<config, hdr, meta, events, client>>
\* Client: SelectFinal over the success keys before stream construction.
Decode == /\ client.kind = "none" /\ published # <<>> /\ (failed \/ steps = MaxSteps)
          /\ LET st == published[1] IN
             client' = IF st \in Domain /\ (Policy # "client-200" \/ st = 200)
                       THEN [status |-> st, events |-> events, kind |-> "stream"]
                       ELSE [status |-> st, events |-> 0, kind |-> "error"]
          /\ UNCHANGED <<config, hdr, meta, opened, published, events, steps, failed>>
Next == SetResponse \/ Trigger \/ Fail \/ Return \/ Decode \/ UNCHANGED vars
Spec == Init /\ [][Next]_vars

Success == published # <<>> /\ published[1] \in Domain
\* A singleton exact success always publishes its own literal status.
ExactLiteral == Literal /\ published # <<>> /\ published[1] # ServiceError /\ HeaderOK =>
                  published[1] = LiteralStatus
\* Status mode never commits a status the handshake metadata did not select.
NoGuessedStatus == ~Literal /\ Success => published[1] = meta.status
\* A required handshake header is present on every committed success.
RequiredHeaderPresent == Success /\ hdr = "required" => meta.header
SingleFinal == Len(published) <= 1
\* A Loom client sees the committed success status, even with zero events.
ClientAgreement == client.kind # "none" /\ Success =>
                     client.kind = "stream" /\ client.status = published[1]
=============================================================================
