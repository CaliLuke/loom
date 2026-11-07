-------------------------- MODULE SSEHandlerFailure --------------------------
\* One generated HTTP SSE handler whose endpoint returns an error, one shared
\* SSEStreamWriter and one retained or in-flight stream caller with at most two
\* writer operations. The current design reads Started, then capture commitment,
\* then encodes the ordinary error. The candidate replaces only the Started read
\* with an admission-only ClaimFailure: one writer critical section that
\* snapshots started and sets closed. Observation, panics and callback timing are
\* outside this model; see README.md for the Go obligations that cover them.
EXTENDS Naturals, Sequences, FiniteSets

CONSTANTS Design, Seeds, Ops

Designs == {"current", "claim", "second-started", "split-claim", "cause-overwrite"}
AllSeeds == {"fresh", "opened", "streamed", "flushFailed", "writeFailed",
             "silentClosed", "completed"}
AllOps == {"open", "write", "complete", "completeNil", "close"}
\* writeEventLocked outcomes after admission. "ok" also stands for post-frame
\* cancellation: the frame is written, nothing is flushed, failed or closed.
WriteOutcomes == {"ok", "untouched", "touched", "flushFail"}
MaxOps == 2
Phases == {"svc", "ret", "closing", "read", "reread", "enc1", "check", "enc2",
           "cleanup", "done"}
Snapshots == {"unset", "open", "unopened"}

ASSUME /\ Design \in Designs
       /\ Seeds \subseteq AllSeeds /\ Seeds # {}
       /\ Ops \subseteq AllOps /\ Ops # {}

VARIABLES
  lc,           \* handler phase
  arg,          \* committed argument passed to HandlerLifecycle.HandlerFailed
  arg2,         \* result of a repeated claim
  claimed,      \* the handler has closed writer admission
  route,        \* HandlerFailed routing: ordinary error encoding or late failure
  started,      \* SSEStreamWriter.started
  closed,       \* SSEStreamWriter.closed (admission only)
  failErr,      \* SSEStreamWriter.failErr cause
  attempts,     \* final statuses attempted on the response, in order
  body,         \* body tokens written to the response, in order
  spc,          \* stream caller phase; "held" means it owns the writer lock
  sop,          \* stream caller's current operation
  sdone,        \* stream caller operations finished
  invokedAt,    \* handler phase when the current operation was invoked
  pubInvokedAt  \* invokedAt of the operation that wrote the latest SSE frame

lifeVars == <<lc, arg, arg2, claimed, route>>
writerVars == <<started, closed, failErr>>
wireVars == <<attempts, body>>
senderVars == <<spc, sop, sdone, invokedAt, pubInvokedAt>>
vars == <<lifeVars, writerVars, wireVars, senderVars>>

Has(s, x) == \E i \in 1..Len(s): s[i] = x
Count(s, x) == Cardinality({i \in 1..Len(s): s[i] = x})
\* ResponseCapture.StatusCode: first final status; net/http keeps the first too.
Cap == IF attempts = <<>> THEN 0 ELSE attempts[1]
Held == spc = "held"
Snapshot == IF started THEN "open" ELSE "unopened"
Mixed == Has(body, "err") /\ Has(body, "sse")

\* Writer state at service return, seeded together with its wire effects.
\* fresh: also a failed Complete preparation or pre-lock encoding failure.
\* opened: Open, or an untouched WriteEvent callback error (initHeaders ran).
\* streamed: a written frame, including post-frame cancellation.
Seed(s) ==
  CASE s = "fresh"        -> <<FALSE, FALSE, "none", <<>>,    <<>>>>
    [] s = "opened"       -> <<TRUE,  FALSE, "none", <<200>>, <<>>>>
    [] s = "streamed"     -> <<TRUE,  FALSE, "none", <<200>>, <<"sse">>>>
    [] s = "flushFailed"  -> <<TRUE,  TRUE,  "io",   <<200>>, <<>>>>
    [] s = "writeFailed"  -> <<TRUE,  TRUE,  "io",   <<200>>, <<"sse">>>>
    [] s = "silentClosed" -> <<FALSE, TRUE,  "none", <<>>,    <<>>>>
    [] s = "completed"    -> <<TRUE,  TRUE,  "none", <<200>>, <<"sse">>>>

Init ==
  /\ \E s \in Seeds:
       /\ started = Seed(s)[1]
       /\ closed = Seed(s)[2]
       /\ failErr = Seed(s)[3]
       /\ attempts = Seed(s)[4]
       /\ body = Seed(s)[5]
  /\ lc = "svc" /\ arg = "unset" /\ arg2 = "unset"
  /\ claimed = FALSE /\ route = "none"
  /\ spc = "idle" /\ sop = "none" /\ sdone = 0
  /\ invokedAt = "none" /\ pubInvokedAt = "none"

(***************************** Handler ******************************)

ServiceFails ==
  /\ lc = "svc"
  /\ lc' = "ret"
  /\ UNCHANGED <<arg, arg2, claimed, route, writerVars, wireVars, senderVars>>

\* Current generated stream.started(): SSEStreamWriter.Started under the lock.
StartedRead ==
  /\ lc = "ret" /\ ~Held
  /\ Design \in {"current", "second-started", "split-claim"}
  /\ arg' = Snapshot
  /\ lc' = IF Design = "split-claim" THEN "closing" ELSE "read"
  /\ UNCHANGED <<arg2, claimed, route, writerVars, wireVars, senderVars>>

\* Candidate ClaimFailure: one critical section, no context check, no cause.
\* cause-overwrite is the rejected variant that records the handler error.
Claim ==
  /\ lc = "ret" /\ ~Held
  /\ Design \in {"claim", "cause-overwrite"}
  /\ arg' = Snapshot
  /\ claimed' = TRUE
  /\ closed' = TRUE
  /\ failErr' = IF Design = "cause-overwrite" THEN "handler" ELSE failErr
  /\ lc' = "read"
  /\ UNCHANGED <<arg2, route, started, wireVars, senderVars>>

\* Rejected split-claim: closes in a second critical section after the read.
SplitClose ==
  /\ lc = "closing" /\ ~Held
  /\ closed' = TRUE
  /\ claimed' = TRUE
  /\ lc' = "read"
  /\ UNCHANGED <<arg, arg2, route, started, failErr, wireVars, senderVars>>

\* HandlerFailed: committed || responseCommitted(). No writer lock.
CaptureRead ==
  /\ lc = "read"
  /\ IF arg = "open" \/ Cap # 0
       THEN route' = "late" /\ lc' = "cleanup"
       ELSE /\ route' = "ordinary"
            /\ lc' = IF Design = "second-started" THEN "reread" ELSE "enc1"
  /\ UNCHANGED <<arg, arg2, claimed, writerVars, wireVars, senderVars>>

\* Rejected second-started: one more Started read before encoding.
SecondStartedRead ==
  /\ lc = "reread" /\ ~Held
  /\ IF started
       THEN route' = "late" /\ lc' = "cleanup"
       ELSE route' = route /\ lc' = "enc1"
  /\ UNCHANGED <<arg, arg2, claimed, writerVars, wireVars, senderVars>>

\* encodeError, first encoder call: commits 400 with an error body, or fails
\* before any commitment.
EncodeFirst ==
  /\ lc = "enc1"
  /\ \/ /\ attempts' = Append(attempts, 400)
        /\ body' = Append(body, "err")
        /\ lc' = "cleanup"
     \/ /\ lc' = "check"
        /\ UNCHANGED wireVars
  /\ UNCHANGED <<arg, arg2, claimed, route, writerVars, senderVars>>

\* encodeError: committed after the failure -> handleFailure, else fallback.
CheckCommitted ==
  /\ lc = "check"
  /\ lc' = IF Cap # 0 THEN "cleanup" ELSE "enc2"
  /\ UNCHANGED <<arg, arg2, claimed, route, writerVars, wireVars, senderVars>>

\* Same ordinary-error owner encodes the encoding error; failure -> handleFailure.
EncodeFallback ==
  /\ lc = "enc2"
  /\ \/ /\ attempts' = Append(attempts, 500)
        /\ body' = Append(body, "err")
     \/ UNCHANGED wireVars
  /\ lc' = "cleanup"
  /\ UNCHANGED <<arg, arg2, claimed, route, writerVars, senderVars>>

\* Deferred SSEStreamWriter.End: cleanup only, never opens or writes.
End ==
  /\ lc = "cleanup" /\ ~Held
  /\ closed' = TRUE
  /\ lc' = "done"
  /\ UNCHANGED <<arg, arg2, claimed, route, started, failErr, wireVars, senderVars>>

\* A repeated ClaimFailure at any later handler phase.
Reclaim ==
  /\ Design = "claim" /\ claimed /\ arg2 = "unset" /\ ~Held
  /\ arg2' = Snapshot
  /\ closed' = TRUE
  /\ UNCHANGED <<lc, arg, claimed, route, started, failErr, wireVars, senderVars>>

(************************** Stream caller ***************************)

Finish ==
  /\ spc' = "idle"
  /\ sop' = "none"
  /\ sdone' = sdone + 1
  /\ invokedAt' = "none"

\* write: generated Send/SendWithContext or SendComment, with pre-lock work.
Invoke ==
  /\ spc = "idle" /\ sdone < MaxOps
  /\ \E o \in Ops:
       /\ sop' = o
       /\ spc' = IF o = "write" THEN "pre" ELSE "adm"
  /\ invokedAt' = lc
  /\ UNCHANGED <<lifeVars, writerVars, wireVars, sdone, pubInvokedAt>>

\* Pre-lock EncodeSSEData or comment validation; failure returns untouched.
Prepare ==
  /\ spc = "pre"
  /\ \/ /\ spc' = "adm"
        /\ UNCHANGED <<sop, sdone, invokedAt>>
     \/ Finish
  /\ UNCHANGED <<lifeVars, writerVars, wireVars, pubInvokedAt>>

\* Open and WriteEvent check context then closedErr; Complete checks closedErr.
\* A refusal changes no writer or wire state.
Admit ==
  /\ spc = "adm" /\ sop \in {"open", "write", "complete"}
  /\ \/ /\ ~closed
        /\ spc' = "held"
        /\ UNCHANGED <<sop, sdone, invokedAt>>
     \/ /\ (closed \/ sop \in {"open", "write"})
        /\ Finish
  /\ UNCHANGED <<lifeVars, writerVars, wireVars, pubInvokedAt>>

\* Open after admission: initHeaders then flush; a flush failure calls fail.
PublishOpen ==
  /\ Held /\ sop = "open"
  /\ IF started
       THEN UNCHANGED <<writerVars, wireVars>>
       ELSE /\ started' = TRUE
            /\ attempts' = Append(attempts, 200)
            /\ \E flushFails \in BOOLEAN:
                 /\ closed' = (closed \/ flushFails)
                 /\ failErr' = IF flushFails /\ failErr = "none" THEN "io" ELSE failErr
            /\ UNCHANGED body
  /\ Finish
  /\ UNCHANGED <<lifeVars, pubInvokedAt>>

\* writeEventLocked after its guards: initHeaders first, then the callback.
Frame(o, terminal) ==
  /\ started' = TRUE
  /\ attempts' = IF started THEN attempts ELSE Append(attempts, 200)
  /\ body' = IF o = "untouched" THEN body ELSE Append(body, "sse")
  /\ failErr' = IF o \in {"touched", "flushFail"} /\ failErr = "none" THEN "io" ELSE failErr
  /\ closed' = (closed \/ terminal \/ o \in {"touched", "flushFail"})
  /\ pubInvokedAt' = IF o = "untouched" THEN pubInvokedAt ELSE invokedAt

PublishWrite ==
  /\ Held /\ sop = "write"
  /\ \E o \in WriteOutcomes: Frame(o, FALSE)
  /\ Finish
  /\ UNCHANGED lifeVars

\* Complete(message): preparation error leaves the writer usable; a canceled
\* context returns before initHeaders; Complete then sets closed in every
\* admitted non-preparation-error case.
PublishComplete ==
  /\ Held /\ sop = "complete"
  /\ \/ UNCHANGED <<writerVars, wireVars, pubInvokedAt>>
     \/ /\ closed' = TRUE
        /\ UNCHANGED <<started, failErr, wireVars, pubInvokedAt>>
     \/ \E o \in WriteOutcomes: Frame(o, TRUE)
  /\ Finish
  /\ UNCHANGED lifeVars

\* Close and Complete(nil): silent closure; Complete(nil) on a closed writer
\* returns an error with the same resulting state.
CloseOp ==
  /\ spc = "adm" /\ sop \in {"close", "completeNil"}
  /\ closed' = TRUE
  /\ Finish
  /\ UNCHANGED <<lifeVars, started, failErr, wireVars, pubInvokedAt>>

\* Legal terminal stuttering; any other deadlock is a model error.
Terminated ==
  /\ lc = "done" /\ spc = "idle" /\ sdone = MaxOps
  /\ UNCHANGED vars

Next ==
  \/ ServiceFails \/ StartedRead \/ Claim \/ SplitClose \/ CaptureRead
  \/ SecondStartedRead \/ EncodeFirst \/ CheckCommitted \/ EncodeFallback
  \/ End \/ Reclaim
  \/ Invoke \/ Prepare \/ Admit \/ PublishOpen \/ PublishWrite
  \/ PublishComplete \/ CloseOp
  \/ Terminated

Spec == Init /\ [][Next]_vars

(**************************** Invariants ****************************)

TypeOK ==
  /\ lc \in Phases
  /\ arg \in Snapshots /\ arg2 \in Snapshots
  /\ claimed \in BOOLEAN
  /\ route \in {"none", "ordinary", "late"}
  /\ started \in BOOLEAN /\ closed \in BOOLEAN
  /\ failErr \in {"none", "io", "handler"}
  /\ attempts \in Seq({200, 400, 500})
  /\ body \in Seq({"sse", "err"})
  /\ spc \in {"idle", "pre", "adm", "held"}
  /\ sop \in AllOps \cup {"none"}
  /\ sdone \in 0..MaxOps
  /\ invokedAt \in Phases \cup {"none"}
  /\ pubInvokedAt \in Phases \cup {"none"}

\* Writer state agrees with its own wire effects, including seeded winners.
WireConsistent ==
  /\ started <=> Has(attempts, 200)
  /\ failErr = "io" => started /\ closed
  /\ failErr # "none" => closed

NoMixedBody == ~Mixed
\* Local: final statuses attempted by this handler and writer, not a protocol claim.
SingleFinalAttempt == Len(attempts) <= 1
OrdinaryNeverOpens == route = "ordinary" => ~started
\* Valid only because no response writer other than the SSE writer acts before routing.
LateOnlyIfOpened == route = "late" => started
\* The claim's returned snapshot stays equal to the writer's started flag.
StableClaim == claimed => arg = Snapshot
ReclaimStable == arg2 \in {"unset", arg}
NoHandlerCause == failErr # "handler"

\* Witness filters: each is expected to fail where the README says so.
\* Executed Go seam witness: 400 and error prefix, then a comment invoked while
\* the encoder was still running; the writer is neither closed nor failed.
ObservedWitnessAbsent ==
  ~(/\ attempts = <<400, 200>>
    /\ body = <<"err", "sse">>
    /\ pubInvokedAt = "cleanup"
    /\ ~closed
    /\ failErr = "none")
\* Model-only: an operation invoked during the service publishes into a mix.
InflightWitnessAbsent == ~(Mixed /\ pubInvokedAt = "svc")
\* Reachability: the ordinary owner's precommit fallback after a claim.
FallbackUnreachable == ~(claimed /\ attempts = <<500>> /\ body = <<"err">>)

(**************************** Properties ****************************)

CausePreserved == [][failErr # "none" => failErr' = failErr]_failErr
ClosedMonotonic == [][closed => closed']_closed
StartedMonotonic == [][started => started']_started
NoPublicationAfterClaim ==
  [][claimed => /\ Count(attempts', 200) = Count(attempts, 200)
                /\ Count(body', "sse") = Count(body, "sse")]_wireVars
=============================================================================
