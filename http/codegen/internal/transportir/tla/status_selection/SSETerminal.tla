---------------------------- MODULE SSETerminal ----------------------------
\* Companion to SSEOpen: one shared admission owner, two contending operations.
\* Current mode faithfully retains ClaimFailure, silent Close/nil Complete,
\* cleanup-only End, context-first Open and terminal-first message preparation.
\* The abstract finalize operation in current mode represents successful handler
\* return followed by End; no current Go FinalizeSuccess API is implied.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANTS Policy, Operations, Seeds, Contexts, InitialMetadata
VARIABLES seed, cancelled, metadata, opened, terminal, cause, wire, frames,
          owner, pc, op, result, before, callbacks, claims, successes, updates,
          suppressionAdmissions, suppressionMarkers
Actors == {1, 2}
Terminals == {"live", "success", "suppressed", "claimed", "failure", "cleanup"}
vars == <<seed, cancelled, metadata, opened, terminal, cause, wire, frames,
          owner, pc, op, result, before, callbacks, claims, successes, updates,
          suppressionAdmissions, suppressionMarkers>>
Shared == <<metadata, opened, terminal, cause, wire, frames>>
History == <<callbacks, claims, successes, updates, suppressionAdmissions, suppressionMarkers>>
Current == Policy = "current"
SuccessOps == {"close", "nil", "complete", "finalize"}
Writes == {"open", "event", "iofail"}
Closing == {"close", "nil", "complete", "finalize", "suppress", "claim", "end"}
Status == IF Current THEN 200 ELSE metadata
CanPublish == Status \in {200, 201}
InitialOpened == seed \in {"opened", "failed", "completed"}
InitialTerm == CASE seed = "failed" -> "failure"
                [] seed = "completed" -> "success"
                [] seed = "claimed" -> "claimed"
                [] seed = "suppressed" -> "suppressed"
                [] OTHER -> "live"
InitialCause == IF seed = "failed" THEN "io" ELSE "none"
Snapshot == [metadata |-> metadata, opened |-> opened, terminal |-> terminal,
             cause |-> cause, wire |-> wire, frames |-> frames]
Init == /\ seed \in Seeds /\ cancelled \in Contexts
        /\ metadata \in (IF Current \/ InitialOpened THEN {200} ELSE InitialMetadata)
        /\ opened = InitialOpened /\ terminal = InitialTerm /\ cause = InitialCause
        /\ wire = (IF InitialOpened THEN <<200>> ELSE <<>>) /\ frames = 0
        /\ owner = 0 /\ pc = [a \in Actors |-> "idle"]
        /\ op = [a \in Actors |-> "none"] /\ result = [a \in Actors |-> "none"]
        /\ before = [a \in Actors |-> [metadata |-> 0, opened |-> FALSE,
               terminal |-> "live", cause |-> "none", wire |-> <<>>, frames |-> 0]]
        /\ callbacks = <<>> /\ claims = <<>> /\ successes = <<>> /\ updates = <<>>
        /\ suppressionAdmissions = <<>> /\ suppressionMarkers = <<>>
Fixed == <<seed, cancelled>>

Acquire(a) == /\ pc[a] = "idle" /\ owner = 0
              /\ \E operation \in Operations : op' = [op EXCEPT ![a] = operation]
              /\ owner' = a /\ pc' = [pc EXCEPT ![a] = "held"]
              /\ before' = [before EXCEPT ![a] = Snapshot]
              /\ UNCHANGED <<Fixed, Shared, result, History>>

\* Linearization occurs while one writer lock is held. No user observation is
\* delivered here. Failed preparation is a separate action so a contender must
\* wait, yet the stream is left usable when this admitted callback fails.
PrepareFailure(a) == /\ owner = a /\ pc[a] = "held" /\ op[a] = "prepareFail"
                     /\ result' = [result EXCEPT ![a] = IF terminal = "live" THEN "prepare-error" ELSE "closed"]
                     /\ pc' = [pc EXCEPT ![a] = "release"]
                     /\ UNCHANGED <<Fixed, Shared, owner, op, before, History>>

\* Existing ClaimFailure ignores cancellation and preserves cause, physical
\* status and prior terminal outcome. It never queues an observer callback.
Claim(a) == /\ owner = a /\ pc[a] = "held" /\ op[a] = "claim"
            /\ terminal' = IF terminal = "live" THEN "claimed" ELSE terminal
            /\ result' = [result EXCEPT ![a] = IF opened THEN "started" ELSE "unstarted"]
            /\ claims' = Append(claims, [before |-> Snapshot, after |->
                 [metadata |-> metadata, opened |-> opened, terminal |-> terminal',
                  cause |-> cause, wire |-> wire, frames |-> frames]])
            /\ pc' = [pc EXCEPT ![a] = "release"]
            /\ UNCHANGED <<Fixed, metadata, opened, cause, wire, frames,
                           owner, op, before, callbacks, successes, updates,
                           suppressionAdmissions, suppressionMarkers>>

\* End is cleanup even on successful or panicking unwinding; publication is a
\* separate explicit operation. Suppression is protocol intent, not nil Complete.
Silent(a) == /\ owner = a /\ pc[a] = "held" /\ op[a] \in {"end", "suppress"}
             /\ terminal' = IF terminal # "live" THEN terminal
                            ELSE IF op[a] = "end" THEN "cleanup" ELSE "suppressed"
             /\ result' = [result EXCEPT ![a] = IF op[a] = "suppress" /\ terminal # "live" THEN "closed" ELSE "ok"]
             /\ suppressionAdmissions' = IF op[a] = "suppress" /\ terminal = "live"
                                           THEN Append(suppressionAdmissions, a) ELSE suppressionAdmissions
             /\ pc' = [pc EXCEPT ![a] = "release"]
             /\ UNCHANGED <<Fixed, metadata, opened, cause, wire, frames,
                            owner, op, before, callbacks, claims, successes, updates, suppressionMarkers>>

\* The generated setter's complete validated snapshot is installed under the
\* same owner as first open. No wrapper metadata lock/check-then-act is modeled.
SetMetadata(a) == /\ owner = a /\ pc[a] = "held" /\ op[a] \in {"set", "setInvalid"}
                  /\ LET admitted == ((~opened /\ terminal = "live") \/ Policy = "late-metadata") /\ op[a] = "set" IN
                       /\ metadata' = IF admitted THEN 201 ELSE metadata
                       /\ result' = [result EXCEPT ![a] = IF admitted THEN "ok" ELSE IF op[a] = "setInvalid" /\ ~opened /\ terminal = "live" THEN "invalid-metadata" ELSE "committed"]
                       /\ updates' = Append(updates, [before |-> Snapshot, value |-> metadata', admitted |-> admitted])
                  /\ pc' = [pc EXCEPT ![a] = "release"]
                  /\ UNCHANGED <<Fixed, opened, terminal, cause, wire, frames,
                                 owner, op, before, callbacks, claims, successes, suppressionAdmissions, suppressionMarkers>>

\* Context precedes closedErr for Open/event, as in the current Go owner.
\* Completion first checks terminal admission; message preparation is assumed
\* to succeed here (the failure case is PrepareFailure), then checks context.
\* Close/finalize idempotently retain a prior terminal outcome. The result tag ok-noop is nil, including a previously failed terminal
\* outcome; its stored cause remains observable on rejected writes/Complete.
\* Suppress below preserves current nil-Complete cancellation behavior.
Operate(a) ==
 /\ owner = a /\ pc[a] = "held" /\ op[a] \in Writes \cup SuccessOps
 /\ LET oldSilent == Current /\ op[a] \in {"close", "nil", "finalize"}
        contextFirst == op[a] \in Writes
        prior == terminal # "live" /\ Policy # "overwrite-terminal"
        repeatSuccess == op[a] \in {"close", "finalize"} /\ prior
        refusal == IF contextFirst /\ cancelled THEN "context"
                   ELSE IF prior THEN IF repeatSuccess THEN "ok-noop" ELSE "closed"
                   ELSE IF ~oldSilent /\ cancelled THEN "context"
                   ELSE "none"
        active == refusal = "none"
        publication == active /\ ~oldSilent /\ ~opened
        selectionError == publication /\ ~CanPublish
        ioError == active /\ ~oldSilent /\ ~selectionError /\ op[a] = "iofail"
        succeeded == active /\ ~selectionError /\ ~ioError /\ op[a] \in SuccessOps
    IN
      /\ opened' = (opened \/ (publication /\ ~selectionError))
      /\ wire' = IF publication /\ ~selectionError THEN Append(wire, Status) ELSE wire
      /\ frames' = IF active /\ ~selectionError /\ ~oldSilent /\ op[a] \in {"event", "complete", "iofail"}
                    THEN frames + 1 ELSE frames
      /\ terminal' = IF ~active THEN
                         IF ~prior /\ op[a] \in {"complete", "nil", "close", "finalize"} THEN "failure" ELSE terminal
                      ELSE IF ioError THEN "failure"
                      ELSE IF selectionError THEN IF op[a] \in Writes THEN terminal ELSE "failure"
                      ELSE IF succeeded THEN "success" ELSE terminal
      /\ cause' = IF cause # "none" THEN cause
                  ELSE IF ioError THEN "io" ELSE cause
      /\ result' = [result EXCEPT ![a] =
            IF ~active THEN refusal ELSE IF selectionError THEN "selection-error"
            ELSE IF ioError THEN "io-error" ELSE "ok"]
      /\ successes' = IF succeeded THEN Append(successes,
               [operation |-> op[a], opened |-> opened', wire |-> wire', metadata |-> metadata]) ELSE successes
 /\ pc' = [pc EXCEPT ![a] = "release"]
 /\ UNCHANGED <<Fixed, metadata, owner, op, before, callbacks, claims, updates, suppressionAdmissions, suppressionMarkers>>

Release(a) == /\ owner = a /\ pc[a] = "release"
              /\ owner' = 0
              /\ pc' = [pc EXCEPT ![a] = IF op[a] = "claim" THEN "done" ELSE "observe"]
              /\ UNCHANGED <<Fixed, Shared, op, result, before, History>>
\* Marker history is specification-only. The wrapper calls Suppressed only
\* after Suppress returned nil. Release-before-Observe structurally assumes
\* delivery outside the calling actor's owner; concrete Go/race tests must
\* establish that correspondence. Observation may be delayed indefinitely;
\* no fairness or callback termination is assumed.
Observe(a) == /\ pc[a] = "observe" /\ owner # a
              /\ callbacks' = Append(callbacks, [actor |-> a, heldBy |-> owner])
              /\ suppressionMarkers' = IF op[a] = "suppress" /\
                     (result[a] = "ok" \/ Policy = "wrong-suppression-wrapper") /\
                     Policy # "missing-suppression-marker"
                   THEN Append(suppressionMarkers, [actor |-> a, returned |-> result[a], heldBy |-> owner])
                   ELSE suppressionMarkers
              /\ pc' = [pc EXCEPT ![a] = "done"]
              /\ UNCHANGED <<Fixed, Shared, owner, op, result, before, claims, successes, updates, suppressionAdmissions>>
Next == \E a \in Actors : Acquire(a) \/ PrepareFailure(a) \/ Claim(a) \/ Silent(a)
                             \/ SetMetadata(a) \/ Operate(a) \/ Release(a) \/ Observe(a)
Spec == Init /\ [][Next]_vars

TypeOK == /\ Len(suppressionAdmissions) <= 2 /\ Len(suppressionMarkers) <= 2
          /\ terminal \in Terminals /\ cause \in {"none", "io"}
          /\ owner \in Actors \cup {0} /\ metadata \in {0, 200, 201}
          /\ frames <= 2 /\ Len(wire) <= 1
SinglePublication == Len(wire) <= 1
SuccessfulPublication == \A i \in 1..Len(successes) : successes[i].opened /\ Len(successes[i].wire) = 1
ClaimPreserves == \A i \in 1..Len(claims) :
   /\ claims[i].before.opened = claims[i].after.opened
   /\ claims[i].before.wire = claims[i].after.wire
   /\ claims[i].before.cause = claims[i].after.cause
   /\ claims[i].before.metadata = claims[i].after.metadata
   /\ claims[i].after.terminal # "live"
CausePreserved == InitialCause = "io" => cause = "io"
CallbacksOutsideOwner == \A i \in 1..Len(callbacks) : callbacks[i].actor # callbacks[i].heldBy
NoClaimCallback == \A i \in 1..Len(callbacks) : op[callbacks[i].actor] # "claim"
MetadataAdmission == \A i \in 1..Len(updates) : updates[i].admitted =>
                         ~updates[i].before.opened /\ updates[i].before.terminal = "live"
SelectedPublication == ~Current /\ opened => wire = <<metadata>>
\* Step properties express effects without conflating callback timing or
\* later legitimate progress by the other actor with this operation's result.
ContextPrecedence == \A a \in Actors :
   /\ (op[a] \in Writes /\ cancelled /\ result[a] # "none" => result[a] = "context")
   /\ (op[a] \in {"complete", "nil", "prepareFail", "suppress"} /\ before[a].terminal # "live" /\ result[a] # "none" => result[a] = "closed")
SelectionFailureOwned == \A a \in Actors :
  (owner = a /\ pc[a] = "release" /\ result[a] = "selection-error") =>
     /\ wire = before[a].wire /\ opened = before[a].opened
     /\ terminal = (IF op[a] \in Writes THEN before[a].terminal ELSE "failure")
HandshakeFailureAbsent == ~(terminal = "failure" /\ ~opened /\ wire = <<>> /\
                              (\E a \in Actors : result[a] = "selection-error"))
PreparationRecoveredAbsent == ~(\E a, b \in Actors : a # b /\ result[a] = "prepare-error" /\ op[b] = "complete" /\ result[b] = "ok")
SuppressedRaceNoWinner == ~(terminal = "suppressed" /\ (\E a \in Actors : op[a] = "close" /\ result[a] = "ok-noop"))
CauseStable == [][cause # "none" => cause' = cause]_vars
TerminalStable == [][terminal # "live" => terminal' = terminal]_vars
CleanupSilent == [][\A a \in Actors : (owner = a /\ pc[a] = "held" /\ op[a] \in {"end", "suppress"}) => UNCHANGED <<opened, wire, metadata, cause>>]_vars
PreparationUsable == [][\A a \in Actors : (owner = a /\ pc[a] = "held" /\ op[a] = "prepareFail") => UNCHANGED Shared]_vars
ClaimDoesNotObserve == [][\A a \in Actors : (owner = a /\ pc[a] = "held" /\ pc'[a] = "release" /\ op[a] = "claim") => UNCHANGED callbacks]_vars

\* Safety at completed observation, not a liveness promise that Observe runs.
AdmissionActors == {suppressionAdmissions[i] : i \in 1..Len(suppressionAdmissions)}
MarkersFor(a) == {i \in 1..Len(suppressionMarkers) : suppressionMarkers[i].actor = a}
\* Derived consistency checks: monotone terminal admission and one Observe
\* per actor imply admission/marker uniqueness under authorized delivery.
SuppressionAdmissionUnique == Len(suppressionAdmissions) <= 1
SuppressionMarkersAuthorized == \A i \in 1..Len(suppressionMarkers) :
 /\ suppressionMarkers[i].actor \in AdmissionActors
 /\ suppressionMarkers[i].returned = "ok"
 /\ suppressionMarkers[i].heldBy # suppressionMarkers[i].actor
SuppressionMarkersUnique == Len(suppressionMarkers) <= 1
SuppressionCompletedObservation == \A a \in Actors :
 (op[a] = "suppress" /\ pc[a] = "done") =>
   Cardinality(MarkersFor(a)) = (IF result[a] = "ok" THEN 1 ELSE 0)

SuppressionRepeatAbsent == ~(\E a,b \in Actors : a # b /\
 op[a] = "suppress" /\ op[b] = "suppress" /\ pc[a] = "done" /\ pc[b] = "done" /\
 result[a] = "ok" /\ result[b] = "closed" /\ Len(suppressionMarkers) = 1 /\ wire = <<>>)
SuppressionBeats(operation) == \E a,b \in Actors : a # b /\
 op[a] = "suppress" /\ result[a] = "ok" /\ pc[a] = "done" /\
 op[b] = operation /\ pc[b] = "done" /\
 result[b] = (IF operation = "complete" THEN "closed" ELSE "ok-noop") /\
 Len(suppressionMarkers) = 1 /\ wire = <<>> /\ ~opened
SuppressionCloseAbsent == ~SuppressionBeats("close")
SuppressionFinalizeAbsent == ~SuppressionBeats("finalize")
SuppressionCompleteAbsent == ~SuppressionBeats("complete")
SuppressionCanceledAbsent == ~(cancelled /\ Len(suppressionMarkers) = 1 /\
 terminal = "suppressed" /\ wire = <<>> /\ ~opened)
SuppressionPendingAbsent == ~(Len(suppressionAdmissions) = 1 /\ Len(suppressionMarkers) = 0 /\
 \E a \in Actors : op[a] = "suppress" /\ pc[a] = "observe" /\ result[a] = "ok")
SuppressionLosesAbsent == ~(\E a,b \in Actors : a # b /\
 op[a] = "complete" /\ result[a] = "ok" /\ pc[a] = "done" /\
 op[b] = "suppress" /\ result[b] = "closed" /\ pc[b] = "done" /\
 Len(suppressionMarkers) = 0 /\ wire = <<201>>)
=============================================================================
