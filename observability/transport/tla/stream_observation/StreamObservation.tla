---------------------- MODULE StreamObservation ----------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Design
VARIABLES opened, closed, locked, queue, active, delivered, failures,
          binding, connectionMethod, lateIdentity, suppressed, underLock
observationVars == <<opened, closed, locked, queue, active, delivered, failures,
          binding, connectionMethod, lateIdentity, suppressed, underLock>>
ObservationInit == /\ opened = FALSE /\ closed = FALSE /\ locked = FALSE
        /\ queue = <<>> /\ active = "none" /\ delivered = <<>>
        /\ failures = 0 /\ binding = "none" /\ connectionMethod = "A"
        /\ lateIdentity = "none" /\ suppressed = FALSE /\ underLock = FALSE
Lock == /\ ~locked /\ locked' = TRUE
        /\ UNCHANGED <<opened, closed, queue, active, delivered, failures,
                       binding, connectionMethod, lateIdentity, suppressed, underLock>>
Unlock == /\ locked /\ locked' = FALSE
          /\ UNCHANGED <<opened, closed, queue, active, delivered, failures,
                         binding, connectionMethod, lateIdentity, suppressed, underLock>>
Commit == /\ locked /\ ~opened /\ opened' = TRUE
          /\ queue' = IF Design = "current" THEN queue ELSE Append(queue, "open")
          /\ UNCHANGED <<closed, locked, active, delivered, failures, binding,
                         connectionMethod, lateIdentity, suppressed, underLock>>
Release == /\ locked /\ opened /\ ~closed /\ closed' = TRUE
           /\ IF Design = "split"
                 THEN /\ delivered' = Append(delivered, "close") /\ UNCHANGED queue
                 ELSE /\ queue' = IF Design = "current" THEN queue ELSE Append(queue, "close")
                      /\ UNCHANGED delivered
           /\ UNCHANGED <<opened, locked, active, failures, binding,
                          connectionMethod, lateIdentity, suppressed, underLock>>
Drain == /\ active = "none" /\ Len(queue) > 0
         /\ (~locked \/ Design = "locked")
         /\ active' = Head(queue) /\ queue' = Tail(queue)
         /\ underLock' = locked
         /\ UNCHANGED <<opened, closed, locked, delivered, failures, binding,
                        connectionMethod, lateIdentity, suppressed>>
CallbackReturns == /\ active # "none"
                   /\ delivered' = Append(delivered, active) /\ active' = "none"
                   /\ UNCHANGED <<opened, closed, locked, queue, failures, binding,
                                  connectionMethod, lateIdentity, suppressed, underLock>>
\* Panic also releases the drainer claim; an entered callback is delivered.
CallbackPanics == CallbackReturns
Failure == /\ ~locked /\ opened /\ ~closed /\ failures < 3
           /\ failures' = failures + 1
           /\ queue' = IF Design = "general" THEN Append(queue, "failure") ELSE queue
           /\ UNCHANGED <<opened, closed, locked, active, delivered, binding,
                          connectionMethod, lateIdentity, suppressed, underLock>>
Bind == /\ binding = "none" /\ binding' = connectionMethod
        /\ UNCHANGED <<opened, closed, locked, queue, active, delivered, failures,
                       connectionMethod, lateIdentity, suppressed, underLock>>
NextMethod == /\ binding = "A" /\ connectionMethod = "A"
              /\ connectionMethod' = "B"
              /\ UNCHANGED <<opened, closed, locked, queue, active, delivered,
                             failures, binding, lateIdentity, suppressed, underLock>>
LateSend == /\ binding = "A" /\ connectionMethod = "B" /\ lateIdentity = "none"
            /\ lateIdentity' = IF Design = "shared" THEN connectionMethod ELSE binding
            /\ UNCHANGED <<opened, closed, locked, queue, active, delivered, failures,
                           binding, connectionMethod, suppressed, underLock>>
Suppress == /\ ~suppressed /\ suppressed' = TRUE
            /\ UNCHANGED <<opened, closed, locked, queue, active, delivered, failures,
                           binding, connectionMethod, lateIdentity, underLock>>
ObservationNext == Lock \/ Unlock \/ Commit \/ Release \/ Drain \/ CallbackReturns
        \/ CallbackPanics \/ Failure \/ Bind \/ NextMethod \/ LateSend \/ Suppress
VARIABLES terminal, stages, writer, finals, lateWrite, terminalCallbackLocked
terminalVars == <<terminal, stages, writer, finals, lateWrite, terminalCallbackLocked>>
vars == <<observationVars, terminalVars>>
TerminalDesign == Design \in {"terminal-wrapper", "terminal-split", "terminal-owned"}
TerminalInit == /\ terminal = FALSE /\ stages = [i \in 1..3 |-> "idle"]
                /\ writer = 0 /\ finals = 0 /\ lateWrite = FALSE
                /\ terminalCallbackLocked = FALSE
Admit(i) == /\ stages[i] = "idle" /\ ~terminal /\ writer = 0
            /\ stages' = [stages EXCEPT ![i] = "ready"]
            /\ writer' = IF Design = "terminal-split" THEN 0 ELSE i
            /\ UNCHANGED <<terminal, finals, lateWrite, terminalCallbackLocked>>
PrepareFailure(i) == /\ stages[i] = "ready" /\ ~terminal
                     /\ (writer = i \/ Design = "terminal-split")
                     /\ stages' = [stages EXCEPT ![i] = "done"]
                     /\ writer' = 0
                     /\ UNCHANGED <<terminal, finals, lateWrite, terminalCallbackLocked>>
Write(i) == /\ stages[i] = "ready"
            /\ stages' = [stages EXCEPT ![i] = "written"]
            /\ lateWrite' = (lateWrite \/ terminal)
            /\ terminal' = (terminal \/ i < 3)
            /\ finals' = finals + (IF i < 3 THEN 1 ELSE 0)
            /\ UNCHANGED <<writer, terminalCallbackLocked>>
ReleaseWriter(i) == /\ stages[i] = "written" /\ writer = i
                    /\ writer' = 0
                    /\ UNCHANGED <<terminal, stages, finals, lateWrite, terminalCallbackLocked>>
TerminalCallback(i) == /\ stages[i] = "written"
                       /\ (writer = 0 \/ Design = "terminal-wrapper")
                       /\ terminalCallbackLocked' = (terminalCallbackLocked \/ writer # 0)
                       /\ stages' = [stages EXCEPT ![i] = "done"]
                       /\ UNCHANGED <<terminal, writer, finals, lateWrite>>
TerminalNext == \E i \in 1..3: Admit(i) \/ PrepareFailure(i) \/ Write(i)
                                  \/ ReleaseWriter(i) \/ TerminalCallback(i)
Init == ObservationInit /\ TerminalInit
Next == IF TerminalDesign THEN TerminalNext /\ UNCHANGED observationVars
        ELSE ObservationNext /\ UNCHANGED terminalVars
Spec == Init /\ [][Next]_vars
NoWriteAfterTerminal == ~lateWrite
SingleFinal == finals <= 1
TerminalCallbackOutsideLock == ~terminalCallbackLocked
Count(s, value) == Cardinality({i \in 1..Len(s): s[i] = value})
LifecycleRecorded == opened => Count(queue, "open") + Count(delivered, "open")
                              + (IF active = "open" THEN 1 ELSE 0) = 1
OpenBeforeClose == Count(delivered, "close") = 0 \/ Count(delivered, "open") = 1
BoundedPending == Len(queue) + (IF active = "none" THEN 0 ELSE 1) <= 2
CallbackOutsideLock == ~underLock
IdentityStable == lateIdentity = "none" \/ lateIdentity = "A"
SingleLifecycle == Count(delivered, "open") <= 1 /\ Count(delivered, "close") <= 1
======================================================================
