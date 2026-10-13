------------------------- MODULE SchedulerIntent -------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES s, q, a
vars == <<s, q, a>>
CommandModes == {"command-dedup", "completed-only", "commands-owned"}
OrderingModes == {"split-check", "reset-reuse", "cutoff-reallocate", "ordering-owned"}
IntentInit == s = [phase |-> "new", intent |-> 0, replica |-> 0,
             issued |-> FALSE, running |-> 0, copy |-> FALSE,
             stop |-> 0, confirmed |-> 0, newer |-> FALSE,
             otherRunning |-> TRUE, restarted |-> FALSE, oldStop |-> FALSE,
             publicationFailed |-> FALSE, stopReplyLost |-> FALSE, fence |-> FALSE]
Request == /\ s.phase = "new"
           /\ s' = [s EXCEPT !.phase = "requested",
                      !.intent = IF Mode = "current" THEN 0 ELSE 1]
UnknownPublication == /\ s.phase = "requested"
                      /\ s' = [s EXCEPT !.phase = "publication-unknown",
                                 !.publicationFailed = TRUE]
Dispatch == /\ s.phase = "requested"
            /\ s' = [s EXCEPT !.phase = "dispatched", !.issued = TRUE,
                       !.copy = TRUE]
\* The first copy starts; its caller loses the response. A routed duplicate
\* may survive the first copy's ack, exactly as WorkerEventAck permits.
LostReply == /\ s.phase = "dispatched"
             /\ s' = [s EXCEPT !.phase = "uncertain", !.running = 1]
Observe == /\ s.replica # s.intent
           /\ s' = [s EXCEPT !.replica = s.intent]
StopAll == /\ s.phase = "uncertain"
           /\ LET found == IF Mode \in {"current", "prewrite"}
                            THEN s.replica ELSE s.intent
              IN s' = [s EXCEPT !.phase = "stop-requested",
                          !.stop = found,
                          !.fence = found # 0,
                          !.intent = IF Mode = "enqueue" /\ found # 0
                                     THEN 0 ELSE @]
\* A later Start may publish before an older Stop's completion arrives.
NewRequest == /\ s.phase = "stop-requested" /\ s.stop = 1 /\ ~s.newer
              /\ s' = [s EXCEPT !.newer = TRUE, !.intent = 2]
LostStopReply == /\ s.phase = "stop-requested" /\ ~s.stopReplyLost
                 /\ s' = [s EXCEPT !.stopReplyLost = TRUE]
StopAck == /\ s.phase = "stop-requested" /\ s.stop = 1
           /\ s' = [s EXCEPT !.phase = "stopped", !.running = 0,
                      !.confirmed = 1,
                      !.intent = IF Mode \in {"owned", "ack-only", "start-fence-only"} /\ s.intent # s.stop
                                 THEN @ ELSE 0]
LateCopy == /\ s.phase = "stopped" /\ s.copy
            /\ s' = [s EXCEPT !.copy = FALSE,
                       !.running = IF Mode \in {"owned", "no-cas", "start-fence-only"}
                                      /\ s.fence THEN @ ELSE 1]
\* A fresh Start after confirmed stop is a distinct incarnation. An old Stop
\* must not stop it, even if delivered later through another routing copy.
Restart == /\ s.phase = "stopped"
           /\ s' = [s EXCEPT !.phase = "restarted", !.intent = 3,
                      !.running = 3, !.restarted = TRUE]
OldStopCopy == /\ s.phase = "restarted" /\ ~s.oldStop
               /\ s' = [s EXCEPT !.oldStop = TRUE,
                          !.running = IF Mode = "owned" THEN @ ELSE 0]
IntentNext == UnknownPublication \/ LostStopReply \/ Restart \/ OldStopCopy
        \/ Request \/ Dispatch \/ LostReply \/ Observe \/ StopAll \/ NewRequest
        \/ StopAck \/ LateCopy
\* Additional bounded traces for review R1/R2. Separate scenarios avoid
\* multiplying unrelated replica states; the original intent configurations
\* keep exactly their earlier state space.
OrderingInit == q = [phase |-> "choose", revision |-> 1, operation |-> 1,
                     checkedRevision |-> 0, checkedOperation |-> 0,
                     badIssue |-> FALSE, reset |-> FALSE, issued |-> FALSE,
                     originalCutoff |-> 0, cutoff |-> 0, running |-> 3,
                     replyLost |-> FALSE, fresh |-> FALSE]
ChooseCheck == /\ q.phase = "choose" /\ Mode # "cutoff-reallocate"
               /\ q' = [q EXCEPT !.phase = "check"]
CheckRevision == /\ q.phase = "check"
                 /\ q' = [q EXCEPT !.phase = "checked",
                            !.checkedRevision = q.revision,
                            !.checkedOperation = q.operation]
Supersede == /\ q.phase = "checked" /\ q.revision = 1
             /\ q' = [q EXCEPT !.revision = 2, !.operation = 2]
Reset == /\ q.phase = "checked" /\ q.revision # 0 /\ ~q.reset
         /\ q' = [q EXCEPT !.revision = 0, !.operation = 0, !.reset = TRUE]
Recreate == /\ q.phase = "checked" /\ q.revision = 0
            /\ q' = [q EXCEPT !.revision = IF Mode = "reset-reuse" THEN 1 ELSE 3,
                       !.operation = 3]
Issue == /\ q.phase = "checked"
         /\ (Mode = "split-check" \/ q.revision = q.checkedRevision)
         /\ q' = [q EXCEPT !.phase = "issued", !.issued = TRUE,
                    !.badIssue = q.operation # q.checkedOperation]
\* One operation is recorded atomically with its first cutoff/event. The
\* reply may be lost even though that stop runs and a new Start follows.
ChooseStop == /\ q.phase = "choose" /\ Mode \in {"cutoff-reallocate", "ordering-owned"}
              /\ q' = [q EXCEPT !.phase = "stop-issued", !.issued = TRUE,
                         !.originalCutoff = 5, !.cutoff = 5]
LoseStopReply == /\ q.phase = "stop-issued"
                 /\ q' = [q EXCEPT !.phase = "stop-reply-lost", !.replyLost = TRUE]
ApplyOriginalStop == /\ q.phase = "stop-reply-lost"
                     /\ q' = [q EXCEPT !.phase = "stop-applied", !.running = 0]
FreshStart == /\ q.phase = "stop-applied"
              /\ q' = [q EXCEPT !.phase = "fresh-start", !.running = 7, !.fresh = TRUE]
RetryStop == /\ q.phase = "fresh-start"
             /\ q' = [q EXCEPT !.phase = "retried",
                        !.cutoff = IF Mode = "cutoff-reallocate" THEN 8 ELSE q.originalCutoff]
ApplyRetry == /\ q.phase = "retried"
              /\ q' = [q EXCEPT !.phase = "retry-applied",
                         !.running = IF q.running <= q.cutoff THEN 0 ELSE @]
OrderingNext == ChooseCheck \/ CheckRevision \/ Supersede \/ Reset \/ Recreate \/ Issue
                \/ ChooseStop \/ LoseStopReply \/ ApplyOriginalStop \/ FreshStart
                \/ RetryStop \/ ApplyRetry
\* Authored commands are distinct from reconciliation of stored operations.
\* Deduplication by key/payload loses a later authored Start, including a
\* still-pending old admission already fenced by an external stop.
CommandInit == a = [phase |-> "choose", complete |-> FALSE, revision |-> 1,
                    token |-> 1, cutoff |-> 0, running |-> 1,
                    authored |-> FALSE, retry |-> FALSE]
CompletedThenStopped == /\ a.phase = "choose"
                        /\ a' = [a EXCEPT !.phase = "inactive", !.complete = TRUE,
                                   !.running = 0, !.cutoff = 2]
ObservedClaimThenFailed == /\ a.phase = "choose"
                           /\ a' = [a EXCEPT !.phase = "inactive", !.complete = TRUE,
                                      !.running = 0]
PendingThenFenced == /\ a.phase = "choose"
                    /\ a' = [a EXCEPT !.phase = "inactive", !.running = 0, !.cutoff = 2]
AuthoredStart == /\ a.phase = "inactive"
                /\ LET fresh == Mode = "commands-owned" \/ (Mode = "completed-only" /\ a.complete)
                   IN a' = [a EXCEPT !.phase = "published", !.authored = TRUE,
                             !.revision = IF fresh THEN 2 ELSE @,
                             !.complete = IF fresh THEN FALSE ELSE @,
                             !.token = IF fresh THEN 0 ELSE @]
ReconcileCommand == /\ a.phase = "published"
                    /\ LET eligible == ~a.complete /\ (a.token = 0 \/ a.token > a.cutoff)
                       IN a' = [a EXCEPT !.phase = "reconciled", !.complete = TRUE,
                                 !.running = IF eligible THEN 3 ELSE @,
                                 !.token = IF eligible THEN 3 ELSE @]
ObserveWithoutCommand == /\ a.phase = "inactive"
                         /\ a' = [a EXCEPT !.phase = "reconciled", !.complete = TRUE]
RetryPending == /\ a.phase = "choose"
                /\ a' = [a EXCEPT !.phase = "retried", !.retry = TRUE]
CommandNext == CompletedThenStopped \/ ObservedClaimThenFailed \/ PendingThenFenced
               \/ AuthoredStart \/ ReconcileCommand \/ ObserveWithoutCommand \/ RetryPending
AuthoredCommandIsNew == a.authored => a.revision = 2
AuthoredStartRuns == a.authored /\ a.phase = "reconciled" => a.running = 3
NoUnauthoredRestart == ~a.authored /\ a.phase = "reconciled" => a.running = 0
ReconciliationKeepsIdentity == a.retry => a.revision = 1 /\ a.token = 1
Init == IntentInit /\ OrderingInit /\ CommandInit
Next == IF Mode \in CommandModes
        THEN CommandNext /\ UNCHANGED <<s, q>>
        ELSE IF Mode \in OrderingModes
             THEN OrderingNext /\ UNCHANGED <<s, a>>
             ELSE IntentNext /\ UNCHANGED <<q, a>>
Spec == Init /\ [][Next]_vars
NoSupersededIssue == ~q.badIssue
StableRetryCutoff == q.issued /\ q.originalCutoff # 0 => q.cutoff = q.originalCutoff
RetryPreservesFreshStart == q.fresh => q.running = 7
ResponsibilityBeforeDispatch == s.phase = "dispatched" => s.intent # 0
AuthoritativeStopSet == s.phase = "stop-requested" => s.stop = 1
NoForgetOnEnqueue == s.phase = "stop-requested" => s.intent # 0
NoOldCopyAfterStop == s.confirmed = 1 => s.running # 1
NoNewerIntentRemoval == s.newer /\ ~s.restarted => s.intent = 2
NoDispatchAfterUnknownPublication == s.publicationFailed => ~s.issued
FreshStartSurvivesOldStop == s.restarted => s.running = 3
ProducerIsolation == s.otherRunning
=============================================================================
