-------------------------- MODULE MessageContext --------------------------
EXTENDS Naturals, TLC
CONSTANT Mode
VARIABLES role, scenario, stage, canceled, started, native, operationDone,
          abort, handlerReturned, workerDone, streamClosed, halfClosed,
          replyDelivered, ioCount, observation, followup, lateIO
vars == <<role, scenario, stage, canceled, started, native, operationDone,
          abort, handlerReturned, workerDone, streamClosed, halfClosed,
          replyDelivered, ioCount, observation, followup, lateIO>>

Init == /\ role \in {"server", "client"}
        /\ scenario \in {"blocked", "pre", "halfclose", "natural"}
        /\ stage = "start"
        /\ canceled = (scenario = "pre")
        /\ started = FALSE
        /\ native = "idle"
        /\ operationDone = FALSE
        /\ abort = FALSE
        /\ handlerReturned = FALSE
        /\ workerDone = FALSE
        /\ streamClosed = FALSE
        /\ halfClosed = FALSE
        /\ replyDelivered = FALSE
        /\ ioCount = 0
        /\ observation = "open"
        /\ followup = FALSE /\ lateIO = FALSE

Start == /\ stage = "start"
         /\ stage' = "running"
         /\ started' = TRUE
         /\ IF canceled /\ Mode # "current"
               THEN /\ operationDone' = TRUE
                    /\ native' = "done"
                    /\ ioCount' = ioCount
               ELSE /\ operationDone' = FALSE
                    /\ native' = "blocked"
                    /\ ioCount' = ioCount + 1
         /\ UNCHANGED <<role, scenario, canceled, abort, handlerReturned,
                        workerDone, streamClosed, halfClosed, replyDelivered,
                        observation, followup, lateIO>>
Cancel == /\ stage = "running" /\ scenario = "blocked" /\ ~canceled
          /\ canceled' = TRUE
          /\ UNCHANGED <<role, scenario, stage, started, native, operationDone,
                         abort, handlerReturned, workerDone, streamClosed,
                         halfClosed, replyDelivered, ioCount, observation, followup, lateIO>>
Abort == /\ canceled /\ native = "blocked" /\ ~abort
         /\ Mode \in {"owned", "join-worker", "per-call", "close-cancels", "late-observer", "unguarded"}
         /\ abort' = TRUE
         /\ streamClosed' = IF role = "client" THEN TRUE ELSE streamClosed
         /\ observation' = IF Mode = "late-observer" THEN observation ELSE "failure-recorded"
         /\ UNCHANGED <<role, scenario, stage, canceled, started, native,
                        operationDone, handlerReturned, workerDone, halfClosed,
                        replyDelivered, ioCount, followup, lateIO>>
DetachCall == /\ Mode = "per-call" /\ canceled /\ native = "blocked"
              /\ ~operationDone
              /\ operationDone' = TRUE
              /\ UNCHANGED <<role, scenario, stage, canceled, started, native,
                             abort, handlerReturned, workerDone, streamClosed,
                             halfClosed, replyDelivered, ioCount, observation, followup, lateIO>>
HalfClose == /\ scenario = "halfclose" /\ started /\ ~halfClosed
             /\ halfClosed' = TRUE
             /\ streamClosed' = (Mode = "close-cancels")
             /\ UNCHANGED <<role, scenario, stage, canceled, started, native,
                            operationDone, abort, handlerReturned, workerDone,
                            replyDelivered, ioCount, observation, followup, lateIO>>
NativeCompletes == /\ native = "blocked"
                   /\ streamClosed \/ scenario = "natural"
                         \/ (scenario = "halfclose" /\ halfClosed)
                   /\ native' = "done"
                   /\ operationDone' = TRUE
                   /\ replyDelivered' = (scenario = "halfclose" /\ ~streamClosed)
                   /\ UNCHANGED <<role, scenario, stage, canceled, started,
                                  abort, handlerReturned, workerDone,
                                  streamClosed, halfClosed, ioCount, observation, followup, lateIO>>
WorkerCompletes == /\ operationDone /\ native # "blocked" /\ ~workerDone
                   /\ workerDone' = TRUE
                   /\ UNCHANGED <<role, scenario, stage, canceled, started,
                                  native, operationDone, abort, handlerReturned,
                                  streamClosed, halfClosed, replyDelivered,
                                  ioCount, observation, followup, lateIO>>
ReturnHandler == /\ ~handlerReturned
                 /\ workerDone \/ (abort /\ Mode # "join-worker")
                 /\ handlerReturned' = TRUE
                 /\ streamClosed' = TRUE
                 /\ observation' = IF observation = "failure-recorded"
                                      THEN "failure-finished" ELSE "success-finished"
                 /\ UNCHANGED <<role, scenario, stage, canceled, started,
                                native, operationDone, abort, workerDone,
                                halfClosed, replyDelivered, ioCount, followup, lateIO>>
ObserveCancel == /\ abort /\ Mode = "late-observer" /\ ~handlerReturned
                 /\ observation = "open"
                 /\ observation' = "failure-recorded"
                 /\ UNCHANGED <<role, scenario, stage, canceled, started, native,
                                operationDone, abort, handlerReturned, workerDone,
                                streamClosed, halfClosed, replyDelivered, ioCount,
                                followup, lateIO>>
Followup == /\ abort /\ native = "done" /\ ~followup
            /\ followup' = TRUE
            /\ lateIO' = (Mode = "unguarded")
            /\ UNCHANGED <<role, scenario, stage, canceled, started, native,
                           operationDone, abort, handlerReturned, workerDone,
                           streamClosed, halfClosed, replyDelivered, ioCount,
                           observation>>
Next == Start \/ Cancel \/ Abort \/ DetachCall \/ HalfClose \/ NativeCompletes
        \/ WorkerCompletes \/ ReturnHandler \/ ObserveCancel \/ Followup
Spec == Init /\ [][Next]_vars /\ WF_vars(Start) /\ WF_vars(Cancel)
        /\ WF_vars(Abort) /\ WF_vars(DetachCall) /\ WF_vars(HalfClose)
        /\ WF_vars(NativeCompletes) /\ WF_vars(WorkerCompletes)
        /\ WF_vars(ReturnHandler) /\ WF_vars(ObserveCancel) /\ WF_vars(Followup)
NoPreCanceledIO == scenario = "pre" => ioCount = 0
NoDetachedNativeWork == operationDone => native # "blocked"
HalfClosePreservesReply == scenario = "halfclose" /\ halfClosed /\ ~workerDone
                          => ~streamClosed
CancellationReturns == (canceled /\ started) ~> operationDone
CancellationReclaimsIO == (canceled /\ started) ~> (native # "blocked")
Completion == started ~> workerDone
NoIOAfterTerminal == ~lateIO
NoFalseSuccess == abort => observation # "success-finished"
=============================================================================
