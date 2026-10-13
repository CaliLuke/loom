------------------------ MODULE RequestScope ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Owners == 0..2
Requests == {1, 2}
VARIABLES ids, rootState, refs, buffers, unbuffered, started, logged, flushed, emitted
vars == <<ids, rootState, refs, buffers, unbuffered, started, logged, flushed, emitted>>
Init == /\ ids \in BOOLEAN
        /\ rootState \in {"empty", "buffered", "flushed"}
        /\ refs = [r \in Requests |-> 0]
        /\ buffers = [o \in Owners |-> IF o = 0 /\ rootState = "buffered" THEN {0} ELSE {}]
        /\ unbuffered = [o \in Owners |-> o = 0 /\ rootState = "flushed"]
        /\ started = {} /\ logged = {} /\ flushed = {}
        /\ emitted = [r \in Requests |-> {}]
Start(r) ==
    LET fresh == Mode = "fresh"
        separate == fresh \/ Mode = "clone" \/ ids
        owner == IF separate THEN r ELSE 0
    IN /\ r \notin started
       /\ refs' = [refs EXCEPT ![r] = owner]
       /\ buffers' = IF separate THEN [buffers EXCEPT ![r] = IF fresh THEN {} ELSE buffers[0]] ELSE buffers
       /\ unbuffered' = IF separate THEN [unbuffered EXCEPT ![r] = IF fresh THEN FALSE ELSE unbuffered[0]] ELSE unbuffered
       /\ started' = started \cup {r}
       /\ UNCHANGED <<ids, rootState, logged, flushed, emitted>>
Log(r) == /\ r \in started \ logged
          /\ buffers' = IF unbuffered[refs[r]] THEN buffers ELSE [buffers EXCEPT ![refs[r]] = @ \cup {r}]
          /\ emitted' = IF unbuffered[refs[r]] THEN [emitted EXCEPT ![r] = @ \cup {r}] ELSE emitted
          /\ logged' = logged \cup {r}
          /\ UNCHANGED <<ids, rootState, refs, unbuffered, started, flushed>>
Flush(r) == /\ r \in logged \ flushed
            /\ emitted' = [emitted EXCEPT ![r] = @ \cup buffers[refs[r]]]
            /\ buffers' = [buffers EXCEPT ![refs[r]] = {}]
            /\ unbuffered' = [unbuffered EXCEPT ![refs[r]] = TRUE]
            /\ flushed' = flushed \cup {r}
            /\ UNCHANGED <<ids, rootState, refs, started, logged>>
Next == \E r \in Requests: Start(r) \/ Log(r) \/ Flush(r)
Spec == Init /\ [][Next]_vars
ScopeOwnership == \A r \in started: refs[r] = r
NoForeignEntries == \A r \in started: buffers[refs[r]] \subseteq {r} /\ emitted[r] \subseteq {r}
IndependentBuffering == \A r \in started \ flushed: ~unbuffered[refs[r]]
=============================================================================
