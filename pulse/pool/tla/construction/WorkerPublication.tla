------------------------ MODULE WorkerPublication ------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLE s
vars == <<s>>
Init == s = [phase |-> "constructing", prepared |-> FALSE,
             member |-> FALSE, heartbeat |-> FALSE, fresh |-> FALSE,
             peerMember |-> FALSE, peerHeartbeat |-> FALSE,
             stream |-> FALSE, queued |-> FALSE, pending |-> TRUE,
             running |-> FALSE, handled |-> FALSE, attempt |-> TRUE,
             closing |-> FALSE, closed |-> FALSE, failed |-> FALSE,
             canceled |-> FALSE,
             cleanupConfirmed |-> FALSE]
Prepare == /\ s.phase = "constructing" /\ ~s.prepared
           /\ (Mode # "early" \/ s.heartbeat)
           /\ s' = [s EXCEPT !.prepared = TRUE, !.stream = TRUE]
Member == /\ s.phase = "constructing" /\ ~s.member
          /\ (Mode = "early" \/ s.prepared)
          /\ s' = [s EXCEPT !.member = TRUE]
Heartbeat == /\ s.phase = "constructing" /\ s.member /\ ~s.heartbeat
             /\ s' = [s EXCEPT !.heartbeat = TRUE, !.fresh = TRUE]
ObserveMember == /\ s.peerMember # s.member
                 /\ s' = [s EXCEPT !.peerMember = s.member]
ObserveHeartbeat == /\ s.peerHeartbeat # s.heartbeat
                    /\ s' = [s EXCEPT !.peerHeartbeat = s.heartbeat]
Route == /\ s.peerMember /\ s.peerHeartbeat /\ s.fresh
         /\ s.stream /\ ~s.queued /\ s.pending
         /\ s' = [s EXCEPT !.queued = TRUE]
Activate == /\ s.phase = "constructing" /\ s.prepared /\ s.heartbeat
            /\ (Mode # "owned" \/ (~s.closing /\ ~s.canceled /\ s.fresh))
            /\ s' = [s EXCEPT !.running = TRUE, !.attempt = FALSE,
                              !.phase = "active"]
Handle == /\ s.running /\ s.queued /\ s.fresh
          /\ s' = [s EXCEPT !.handled = TRUE, !.queued = FALSE,
                            !.pending = FALSE]
Fail == /\ s.phase = "constructing"
        /\ s' = [s EXCEPT !.phase = "aborting", !.failed = TRUE]
AbortConfirmed == /\ s.phase = "aborting"
                  /\ s' = [s EXCEPT !.member = FALSE, !.heartbeat = FALSE,
                      !.stream = IF Mode = "withdraw_only" THEN s.stream ELSE FALSE,
                      !.queued = IF Mode = "withdraw_only" THEN s.queued ELSE FALSE,
                      !.pending = IF Mode = "ack_abort" THEN FALSE ELSE s.pending,
                      !.prepared = FALSE, !.attempt = FALSE,
                      !.cleanupConfirmed = TRUE, !.phase = "failed"]
AbortUncertain == /\ s.phase = "aborting"
                  /\ \E removedMember, removedHeartbeat, removedStream \in BOOLEAN:
                    s' = [s EXCEPT
                      !.member = s.member /\ ~removedMember,
                      !.heartbeat = s.heartbeat /\ ~removedHeartbeat,
                      !.stream = s.stream /\ ~removedStream,
                      !.queued = s.queued /\ ~removedStream,
                      !.prepared = FALSE, !.attempt = FALSE, !.phase = "failed"]
Expire == /\ s.fresh /\ s' = [s EXCEPT !.fresh = FALSE]
\* Recovery abstracts eventual delivery to a different healthy worker. The
\* original PEL event remains recoverable even if an earlier route was deleted.
Recover == /\ s.phase = "failed" /\ ~s.fresh /\ s.pending
           /\ s' = [s EXCEPT !.handled = TRUE, !.pending = FALSE, !.queued = FALSE]
Cancel == /\ ~s.canceled /\ s' = [s EXCEPT !.canceled = TRUE]
BeginClose == /\ ~s.closing /\ s' = [s EXCEPT !.closing = TRUE]
FinishClose == /\ s.closing /\ ~s.closed
               /\ (Mode # "owned" \/ ~s.attempt)
               /\ s' = [s EXCEPT !.closed = TRUE, !.running = FALSE]
Next == Prepare \/ Member \/ Heartbeat \/ ObserveMember \/ ObserveHeartbeat
        \/ Route \/ Activate \/ Handle \/ Fail \/ AbortConfirmed \/ AbortUncertain
        \/ Expire \/ Recover \/ Cancel \/ BeginClose \/ FinishClose
Spec == Init /\ [][Next]_vars
PublicationHasPreparation == s.phase = "constructing" /\ s.heartbeat => s.prepared
NoWorkerAfterClose == s.closed => ~s.running
ConfirmedAbortRemovesPrivateStream == s.cleanupConfirmed => ~s.stream
OriginalEventPreserved == s.pending \/ s.handled
FailedAttemptNeverRuns == s.failed => ~s.running
FailedAttemptReleasesLocal == s.phase = "failed" => ~s.prepared /\ ~s.attempt
==========================================================================
