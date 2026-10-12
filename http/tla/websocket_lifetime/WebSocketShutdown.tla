----------------------- MODULE WebSocketShutdown -----------------------
EXTENDS Naturals, TLC

CONSTANT Design
VARIABLES attach, socket, reader, contextDone, phase, cause, firstCause,
          canceled, writerBlocked, control, closeCount, cancelCount

vars == <<attach, socket, reader, contextDone, phase, cause, firstCause,
          canceled, writerBlocked, control, closeCount, cancelCount>>

Init == /\ attach = "none" /\ socket = FALSE /\ reader = "none"
        /\ contextDone = FALSE /\ phase = "open" /\ cause = "none"
        /\ firstCause = "none" /\ canceled = FALSE
        /\ writerBlocked = FALSE /\ control = "none"
        /\ closeCount = 0 /\ cancelCount = 0

BeginHook == /\ attach = "none" /\ attach' = "hook" /\ socket' = TRUE
             /\ UNCHANGED <<reader, contextDone, phase, cause, firstCause,
                            canceled, writerBlocked, control, closeCount,
                            cancelCount>>

FinishHook == /\ attach = "hook" /\ attach' = "owned"
              /\ IF phase = "open" /\ ~contextDone
                    THEN /\ reader' = "running"
                         /\ UNCHANGED <<socket, closeCount>>
                    ELSE /\ reader' = "done" /\ socket' = FALSE
                         /\ closeCount' = closeCount + 1
              /\ UNCHANGED <<contextDone, phase, cause, firstCause,
                             canceled, writerBlocked, control, cancelCount>>

CancelContext == /\ ~contextDone /\ contextDone' = TRUE
                 /\ UNCHANGED <<attach, socket, reader, phase, cause,
                                firstCause, canceled, writerBlocked, control,
                                closeCount, cancelCount>>

PublishStop(why) == /\ phase = "open" /\ phase' = "published"
                   /\ cause' = why /\ firstCause' = why
                   /\ canceled' = TRUE /\ cancelCount' = 1
                   /\ UNCHANGED <<attach, socket, reader, contextDone,
                                  writerBlocked, control, closeCount>>

Stop == PublishStop("explicit") \/ PublishStop("handler")
        \/ (reader = "running" /\ PublishStop("read"))

ContextStop == contextDone /\ PublishStop("context")

BeginWrite == /\ attach = "owned" /\ phase = "open" /\ ~writerBlocked
              /\ writerBlocked' = TRUE
              /\ UNCHANGED <<attach, socket, reader, contextDone, phase,
                             cause, firstCause, canceled, control,
                             closeCount, cancelCount>>

TryControl == /\ phase = "published"
              /\ (Design # "writer_lock" \/ ~writerBlocked)
              /\ phase' = "control" /\ control' \in {"sent", "failed", "timeout"}
              /\ UNCHANGED <<attach, socket, reader, contextDone, cause,
                             firstCause, canceled, writerBlocked,
                             closeCount, cancelCount>>

ReleaseSocket == /\ phase = "control" /\ phase' = "released"
                 /\ IF attach = "owned" /\ socket
                       THEN /\ socket' = FALSE /\ closeCount' = closeCount + 1
                       ELSE UNCHANGED <<socket, closeCount>>
                 /\ UNCHANGED <<attach, reader, contextDone, cause, firstCause,
                                canceled, writerBlocked, control, cancelCount>>

ReaderExit == /\ reader = "running" /\ ~socket /\ reader' = "done"
              /\ UNCHANGED <<attach, socket, contextDone, phase, cause,
                             firstCause, canceled, writerBlocked, control,
                             closeCount, cancelCount>>

WriterExit == /\ writerBlocked /\ ~socket /\ writerBlocked' = FALSE
              /\ UNCHANGED <<attach, socket, reader, contextDone, phase,
                             cause, firstCause, canceled, control,
                             closeCount, cancelCount>>

Next == BeginHook \/ FinishHook \/ CancelContext \/ Stop \/ ContextStop \/ BeginWrite
        \/ TryControl \/ ReleaseSocket \/ ReaderExit \/ WriterExit

Spec == Init /\ [][Next]_vars
        /\ WF_vars(FinishHook) /\ WF_vars(TryControl)
        /\ WF_vars(ReleaseSocket) /\ WF_vars(ReaderExit)
        /\ WF_vars(WriterExit) /\ WF_vars(ContextStop)

FirstCauseStable == cause = firstCause
SingleRelease == closeCount <= 1
SingleCancel == cancelCount <= 1
PublicationCancels == phase # "open" => canceled
NoReaderAfterCanceledHook == attach = "owned" /\ phase # "open"
                            /\ reader = "none" => ~socket
ReleasedOwnsNoSocket == phase = "released" /\ attach = "owned" => ~socket
StopFinishes == phase = "published" ~> phase = "released"
WorkersFinish == phase = "released" /\ attach = "owned" ~>
                reader # "running" /\ ~writerBlocked
CanceledAttachmentCloses == contextDone /\ attach = "owned" ~> ~socket
=============================================================================
