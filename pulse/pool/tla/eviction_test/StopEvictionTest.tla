------------------------- MODULE StopEvictionTest -------------------------
EXTENDS Naturals, TLC
CONSTANT QuiesceFirst
VARIABLES phase, runtime, callback, released, tracked, evicted
vars == <<phase, runtime, callback, released, tracked, evicted>>

Init == /\ phase = IF QuiesceFirst THEN "quiesce" ELSE "publish"
        /\ runtime = TRUE /\ callback = "none" /\ released = FALSE
        /\ tracked = TRUE /\ evicted = FALSE

Quiesce == /\ phase = "quiesce"
           /\ runtime' = FALSE /\ phase' = "publish"
           /\ UNCHANGED <<callback, released, tracked, evicted>>
Publish == /\ phase = "publish" /\ phase' = "callback"
           /\ UNCHANGED <<runtime, callback, released, tracked, evicted>>
Enter(owner) == /\ phase = "callback" /\ callback = "none"
                /\ (owner = "direct" \/ runtime)
                /\ callback' = owner /\ phase' = "join"
                /\ UNCHANGED <<runtime, released, tracked, evicted>>
\* Worker.stop joins the runtime. A callback on that runtime must return first.
Join == /\ phase = "join" /\ callback # "runtime"
        /\ runtime' = FALSE /\ phase' = "release"
        /\ UNCHANGED <<callback, released, tracked, evicted>>
Release == /\ phase = "release" /\ released' = TRUE /\ phase' = "evict"
           /\ UNCHANGED <<runtime, callback, tracked, evicted>>
FailStop == /\ released /\ callback \in {"direct", "runtime"}
            /\ callback' = "failed"
            /\ UNCHANGED <<phase, runtime, released, tracked, evicted>>
Evict == /\ phase = "evict" /\ callback = "failed"
         /\ tracked' = FALSE /\ evicted' = TRUE /\ phase' = "done"
         /\ UNCHANGED <<runtime, callback, released>>
Next == Quiesce \/ Publish \/ Enter("direct") \/ Enter("runtime")
        \/ Join \/ Release \/ FailStop \/ Evict
Spec == Init /\ [][Next]_vars
NoJoinCycle == ~(phase = "join" /\ callback = "runtime" /\ ~released)
FailedStopRetainsJob == callback = "failed" /\ ~evicted => tracked
EvictionForgetsJob == evicted => ~tracked /\ callback = "failed"
=============================================================================
