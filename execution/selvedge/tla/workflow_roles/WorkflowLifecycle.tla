-------------------------- MODULE WorkflowLifecycle --------------------------
\* Role lifecycle gate for #1024: one Producer or Worker role, its native
\* handle, concurrent Start/Shutdown, interrupted Shutdown waits, in-flight
\* Submit/Lookup operations and registration sealing.
\*
\* Upstream = "producerHandle" assumes the requested Selvedge producer-only
\* handle (CaliLuke/selvedge#2). Upstream = "clientApp" is the v0.7.0 App with
\* no registrations.
\*
\* NativeShutdown = "completionWait" assumes the supported cleanup-completion
\* wait requested in selvedge#2: native shutdown reports a cleanup result only
\* after cleanup completes. NativeShutdown = "v070Timer" is the v0.7.0 baseline:
\* a finite internal timer may return an untyped error while cleanup continues,
\* and Loom cannot tell that error from a cleanup result.
EXTENDS Naturals, FiniteSets

CONSTANTS Role, Upstream, NativeShutdown, Mode
ASSUME Role \in {"producer", "worker"}
ASSUME Upstream \in {"producerHandle", "clientApp"}
ASSUME NativeShutdown \in {"completionWait", "v070Timer"}
ASSUME Mode \in {"checked", "exposeApp", "timeoutStops", "noDrain", "callerCtx",
                 "startIgnoresShutdown"}

Waiters == {1, 2}
OpIDs == {1, 2}

VARIABLES state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
          startOut, cleanup, waiters, ops, executed, engineLoops
vars == <<state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
          startOut, cleanup, waiters, ops, executed, engineLoops>>

Inside == {j \in OpIDs : ops[j] = "inside"}
\* Handles that native cleanup must close.
Opened == {"live", "dead"}

Init ==
    /\ state = "new" /\ shutdownReq = FALSE /\ regs = FALSE /\ lateReg = FALSE
    /\ native = "none" /\ initCtx = "live" /\ callerCtx = "live" /\ watcher = "off"
    /\ startOut = "none" /\ cleanup = "none"
    /\ waiters = [i \in Waiters |-> "idle"] /\ ops = [j \in OpIDs |-> "idle"]
    /\ executed = FALSE /\ engineLoops = FALSE

\* Only a Worker has a registration path, and only while new. exposeApp is the
\* rejected alternative that exposes the producer's native App.
Register ==
    /\ \/ Role = "worker" /\ state = "new"
       \/ Mode = "exposeApp"
    /\ regs' = TRUE
    /\ lateReg' = (lateReg \/ state # "new")
    /\ UNCHANGED <<state, shutdownReq, native, initCtx, callerCtx, watcher, startOut,
                   cleanup, waiters, ops, executed, engineLoops>>

\* Start passes native start a Loom-owned context. A watcher cancels it only
\* while initialization runs; callerCtx is the rejected direct pass-through.
StartBegin ==
    /\ state = "new"
    /\ state' = "starting" /\ native' = "init" /\ watcher' = "watching"
    /\ initCtx' = IF Mode = "callerCtx" THEN callerCtx ELSE "live"
    /\ UNCHANGED <<shutdownReq, regs, lateReg, callerCtx, startOut, cleanup, waiters, ops,
                   executed, engineLoops>>

NativeInit ==
    /\ native = "init"
    /\ \/ native' = "failed" /\ UNCHANGED engineLoops
       \/ /\ initCtx = "live" /\ native' = "live"
          \* A v0.7.0 App always runs leader election, resumption and activity loops.
          /\ engineLoops' = (Role = "worker" \/ Upstream = "clientApp")
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, initCtx, callerCtx, watcher, startOut,
                   cleanup, waiters, ops, executed>>

CallerCancel ==
    /\ callerCtx = "live"
    /\ callerCtx' = "cancelled"
    /\ LET dies == watcher = "watching" \/ (Mode = "callerCtx" /\ native # "none")
       IN /\ initCtx' = IF dies THEN "cancelled" ELSE initCtx
          \* A cancelled lifetime stops a started handle's background work.
          /\ native' = IF dies /\ native = "live" THEN "dead" ELSE native
          /\ engineLoops' = IF dies THEN FALSE ELSE engineLoops
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, watcher, startOut, cleanup, waiters,
                   ops, executed>>

\* Stopping the watcher and reading initCtx happen under the gate, so a
\* cancellation is either observed here or can no longer occur.
StartEnd ==
    /\ state = "starting" /\ native \in {"live", "failed", "dead"}
    /\ watcher' = "stopped"
    /\ LET ok == native = "live" /\ initCtx = "live"
       IN CASE shutdownReq /\ Mode # "startIgnoresShutdown" ->
                 /\ state' = "stopping" /\ cleanup' = "draining" /\ startOut' = "err"
            [] ok -> /\ state' = "running" /\ startOut' = "ok" /\ UNCHANGED cleanup
            [] native \in Opened -> /\ state' = "stopping" /\ cleanup' = "draining" /\ startOut' = "err"
            [] OTHER -> /\ state' = "stopped" /\ cleanup' = "done" /\ startOut' = "err"
    /\ UNCHANGED <<shutdownReq, regs, lateReg, native, initCtx, callerCtx, waiters, ops,
                   executed, engineLoops>>

ShutdownCall(i) ==
    /\ waiters[i] = "idle"
    /\ CASE state = "new" ->
              /\ state' = "stopped" /\ cleanup' = "done" /\ waiters' = [waiters EXCEPT ![i] = "result"]
              /\ UNCHANGED shutdownReq
         [] state = "starting" ->
              /\ shutdownReq' = TRUE /\ waiters' = [waiters EXCEPT ![i] = "waiting"]
              /\ UNCHANGED <<state, cleanup>>
         [] state = "running" ->
              /\ state' = "stopping" /\ cleanup' = "draining"
              /\ waiters' = [waiters EXCEPT ![i] = "waiting"] /\ UNCHANGED shutdownReq
         \* stopped means cleanup completed; its recorded result is returned.
         [] state = "stopped" ->
              /\ waiters' = [waiters EXCEPT ![i] = "result"] /\ UNCHANGED <<state, cleanup, shutdownReq>>
         [] OTHER ->
              /\ waiters' = [waiters EXCEPT ![i] = "waiting"] /\ UNCHANGED <<state, cleanup, shutdownReq>>
    /\ UNCHANGED <<regs, lateReg, native, initCtx, callerCtx, watcher, startOut, ops,
                   executed, engineLoops>>

ShutdownReturn(i) ==
    /\ waiters[i] = "waiting" /\ cleanup = "done"
    /\ waiters' = [waiters EXCEPT ![i] = "result"]
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
                   startOut, cleanup, ops, executed, engineLoops>>

\* The caller's context expires: ErrShutdownIncomplete. timeoutStops is the
\* rejected alternative that also marks the role stopped.
ShutdownInterrupt(i) ==
    /\ waiters[i] = "waiting" /\ cleanup # "done"
    /\ waiters' = [waiters EXCEPT ![i] = "incomplete"]
    /\ state' = IF Mode = "timeoutStops" /\ state = "stopping" THEN "stopped" ELSE state
    /\ UNCHANGED <<shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher, startOut,
                   cleanup, ops, executed, engineLoops>>

\* Submit and Lookup are admitted only while running and hold an in-flight count.
OpEnter(j) ==
    /\ ops[j] = "idle"
    /\ ops' = [ops EXCEPT ![j] = IF state = "running" THEN "inside" ELSE "notRunning"]
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
                   startOut, cleanup, waiters, executed, engineLoops>>

OpExit(j) ==
    /\ ops[j] = "inside"
    /\ ops' = [ops EXCEPT ![j] = "done"]
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
                   startOut, cleanup, waiters, executed, engineLoops>>

\* Cleanup drains admitted operations before native shutdown.
CleanupDrain ==
    /\ cleanup = "draining" /\ (Inside = {} \/ Mode = "noDrain")
    /\ cleanup' = "native"
    /\ native' = IF native \in Opened THEN "closing" ELSE native
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, initCtx, callerCtx, watcher, startOut,
                   waiters, ops, executed, engineLoops>>

\* Loom's cleanup waits for the native shutdown result and records it. Under
\* completionWait (assumed native contract) that result exists only after
\* native cleanup closes storage. Under v070Timer the finite timer may fire
\* first; Loom then records an indistinguishable error while storage is closing.
CleanupNative ==
    /\ cleanup = "native"
    /\ cleanup' = "done" /\ state' = "stopped"
    /\ \/ /\ native' = IF native = "closing" THEN "closed" ELSE native
          /\ engineLoops' = FALSE
       \/ /\ NativeShutdown = "v070Timer" /\ native = "closing"
          /\ UNCHANGED <<native, engineLoops>>
    /\ UNCHANGED <<shutdownReq, regs, lateReg, initCtx, callerCtx, watcher, startOut, waiters,
                   ops, executed>>

NativeCloseLate ==
    /\ native = "closing" /\ cleanup = "done"
    /\ native' = "closed" /\ engineLoops' = FALSE
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, initCtx, callerCtx, watcher, startOut,
                   cleanup, waiters, ops, executed>>

\* A live handle runs a registered implementation for an admitted job.
Execute ==
    /\ native = "live" /\ regs /\ ~executed
    /\ executed' = TRUE
    /\ UNCHANGED <<state, shutdownReq, regs, lateReg, native, initCtx, callerCtx, watcher,
                   startOut, cleanup, waiters, ops, engineLoops>>

Next ==
    \/ Register \/ StartBegin \/ NativeInit \/ CallerCancel \/ StartEnd
    \/ \E i \in Waiters : ShutdownCall(i) \/ ShutdownReturn(i) \/ ShutdownInterrupt(i)
    \/ \E j \in OpIDs : OpEnter(j) \/ OpExit(j)
    \/ CleanupDrain \/ CleanupNative \/ NativeCloseLate \/ Execute
Fairness == WF_vars(CleanupDrain) /\ WF_vars(CleanupNative) /\ WF_vars(NativeCloseLate)
            /\ WF_vars(NativeInit) /\ WF_vars(StartEnd) /\ \A j \in OpIDs : WF_vars(OpExit(j))
Spec == Init /\ [][Next]_vars /\ Fairness

\* Invariant P: a producer never executes workflow or activity code.
InvariantP == Role = "producer" => ~executed
\* Prerequisite: a producer handle runs no engine loops (leases, sweeps, dispatch).
ProducerNoEngineLoops == Role = "producer" => ~engineLoops
RegistrationSealed == ~lateReg
RunningUsable == state = "running" => native = "live" /\ initCtx = "live"
ShutdownRequestHonored == ~(shutdownReq /\ state = "running")
ShutdownTruthful == \A i \in Waiters : waiters[i] = "result" => cleanup = "done"
CleanupComplete == cleanup = "done" => native \in {"none", "failed", "closed"}
NoUseAfterClose == Inside = {} \/ native \notin {"closing", "closed"}
StartTruthful == startOut = "ok" => state # "starting"
CleanupTerminates == (state = "stopping") ~> (state = "stopped")
=============================================================================
