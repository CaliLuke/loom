------------------------- MODULE TickerLifecycle -------------------------
EXTENDS Naturals, TLC
CONSTANTS Scenario, CleanupAbort, MissingTerminal, GuardTerminal, GuardRearm, DeleteOnAbort
VARIABLES phase, subscription, childContext, timer, selected, shared,
          peerLive, panicked, notice
vars == <<phase, subscription, childContext, timer, selected, shared,
          peerLive, panicked, notice>>
Init == /\ phase = "acquiring"
        /\ subscription = TRUE /\ childContext = TRUE
        /\ timer = FALSE /\ selected = FALSE
        /\ shared = TRUE /\ peerLive = TRUE /\ panicked = FALSE /\ notice = FALSE

Construct == /\ phase = "acquiring" /\ Scenario # "abort"
             /\ phase' = "live" /\ timer' = TRUE
             /\ UNCHANGED <<subscription, childContext, selected, shared,
                            peerLive, panicked, notice>>
Abort == /\ phase = "acquiring" /\ Scenario \in {"abort", "local", "all"}
         /\ phase' = "terminal"
         /\ subscription' = IF CleanupAbort THEN FALSE ELSE subscription
         /\ childContext' = IF CleanupAbort THEN FALSE ELSE childContext
         /\ shared' = IF DeleteOnAbort THEN FALSE ELSE shared
         /\ UNCHANGED <<timer, selected, peerLive, panicked, notice>>

SelectTick == /\ phase = "live" /\ timer /\ ~selected
              /\ selected' = TRUE /\ timer' = FALSE
              /\ UNCHANGED <<phase, subscription, childContext, shared,
                             peerLive, panicked, notice>>
Close == /\ phase = "live" /\ Scenario \in {"close", "local", "all"}
         /\ phase' = "terminal" /\ subscription' = FALSE
         /\ childContext' = FALSE /\ timer' = FALSE
         /\ UNCHANGED <<selected, shared, peerLive, panicked, notice>>
RemoteStop == /\ phase = "live" /\ shared /\ Scenario \in {"remote", "all"}
              /\ shared' = FALSE /\ peerLive' = FALSE
              /\ UNCHANGED <<phase, subscription, childContext, timer,
                             selected, panicked, notice>>
GlobalStop == /\ phase = "live" /\ Scenario \in {"local", "all"}
              /\ phase' = "terminal" /\ shared' = FALSE /\ peerLive' = FALSE
              /\ subscription' = FALSE /\ childContext' = FALSE /\ timer' = FALSE
              /\ UNCHANGED <<selected, panicked, notice>>

\* Delayed map notification: the current implementation unsubscribes but
\* leaves the child context and timer alive when its goroutine returns.
ObserveDeletion == /\ phase = "live" /\ ~shared
                   /\ phase' = "terminal" /\ subscription' = FALSE
                   /\ childContext' = IF MissingTerminal THEN FALSE ELSE childContext
                   /\ timer' = IF MissingTerminal THEN FALSE ELSE timer
                   /\ UNCHANGED <<selected, shared, peerLive, panicked, notice>>

\* A selected timer case can enter handleTick after Close released its lock.
SkipTerminal == /\ selected /\ phase = "terminal" /\ GuardTerminal
                /\ selected' = FALSE
                /\ UNCHANGED <<phase, subscription, childContext, timer,
                               shared, peerLive, panicked, notice>>
Retry == /\ selected /\ (phase = "live" \/ ~GuardTerminal)
         /\ timer' = TRUE /\ selected' = FALSE
         /\ UNCHANGED <<phase, subscription, childContext, shared, peerLive, panicked, notice>>
Advance == /\ selected /\ phase = "live" /\ shared
           /\ timer' = TRUE /\ selected' = FALSE
           /\ UNCHANGED <<phase, subscription, childContext, shared, peerLive, panicked, notice>>
Missing == /\ selected /\ phase = "live" /\ ~shared
           /\ phase' = "terminal" /\ selected' = FALSE
           /\ panicked' = ~MissingTerminal
           /\ subscription' = IF MissingTerminal THEN FALSE ELSE subscription
           /\ childContext' = IF MissingTerminal THEN FALSE ELSE childContext
           /\ timer' = FALSE
           /\ UNCHANGED <<shared, peerLive, notice>>
\* A map notification can read a present value before taking the ticker lock.
SelectNotice == /\ phase = "live" /\ shared /\ ~notice
                /\ notice' = TRUE
                /\ UNCHANGED <<phase, subscription, childContext, timer,
                               selected, shared, peerLive, panicked>>
ApplyNotice == /\ notice
               /\ notice' = FALSE
               /\ timer' = IF phase = "terminal" /\ GuardRearm THEN timer ELSE TRUE
               /\ UNCHANGED <<phase, subscription, childContext, selected,
                              shared, peerLive, panicked>>
Next == Construct \/ Abort \/ SelectTick \/ Close \/ RemoteStop \/ GlobalStop
        \/ ObserveDeletion \/ SkipTerminal \/ Retry \/ Advance \/ Missing
        \/ SelectNotice \/ ApplyNotice
Spec == Init /\ [][Next]_vars
NoParseOfAbsence == ~panicked
TerminalReleasesLocal == phase = "terminal" => ~subscription /\ ~childContext /\ ~timer
LocalCleanupPreservesPeer == peerLive => shared
=============================================================================
