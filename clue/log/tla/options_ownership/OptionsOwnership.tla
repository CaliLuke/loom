----------------------- MODULE OptionsOwnership -----------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
Loggers == 0..2
Children == {1, 2}
VARIABLES refs, options, writers, configured, expectedDebug, expectedWriter
vars == <<refs, options, writers, configured, expectedDebug, expectedWriter>>
Init == /\ refs = [l \in Loggers |-> 0]
        /\ options = [i \in Loggers |-> [debug |-> FALSE, output |-> 0]]
        /\ writers = [i \in Loggers |-> 0]
        /\ configured = {}
        /\ expectedDebug = [l \in Loggers |-> FALSE]
        /\ expectedWriter = [l \in Loggers |-> 0]
Configure(c, flag) ==
    LET old == options[refs[c]]
        slot == IF Mode = "current" THEN refs[c] ELSE c
        output == IF Mode = "deep" THEN c ELSE old.output
        debug == flag = "debug"
        writer == IF flag = "writer" THEN 1 ELSE writers[old.output]
    IN /\ c \notin configured
       /\ refs' = [refs EXCEPT ![c] = slot]
       /\ options' = [options EXCEPT ![slot] = [debug |-> debug, output |-> output]]
       /\ writers' = [writers EXCEPT ![output] = writer]
       /\ configured' = configured \cup {c}
       /\ expectedDebug' = [expectedDebug EXCEPT ![c] = debug]
       /\ expectedWriter' = [expectedWriter EXCEPT ![c] = writer]
Next == \E c \in Children, flag \in {"debug", "writer"}: Configure(c, flag)
Spec == Init /\ [][Next]_vars
ConfigurationIsolated == \A l \in Loggers:
    /\ options[refs[l]].debug = expectedDebug[l]
    /\ writers[options[refs[l]].output] = expectedWriter[l]
=============================================================================
