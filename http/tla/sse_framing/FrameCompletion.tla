------------------------ MODULE FrameCompletion ------------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES input, available, pos, lineData, blockData, skipLF, complete, emitted, done
vars == <<input, available, pos, lineData, blockData, skipLF, complete, emitted, done>>
Inputs == UNION {[1..n -> {"data", "CR", "LF"}]: n \in 0..5}
Init == /\ input \in Inputs /\ available = 0 /\ pos = 1
        /\ lineData = FALSE /\ blockData = FALSE /\ skipLF = FALSE
        /\ complete = <<>> /\ emitted = <<>> /\ done = FALSE
Deliver == /\ ~done /\ available < Len(input)
           /\ available' \in (available+1)..Len(input)
           /\ UNCHANGED <<input,pos,lineData,blockData,skipLF,complete,emitted,done>>
Scan == /\ ~done /\ pos <= available
        /\ LET token == input[pos]
               ignoredLF == skipLF /\ token = "LF"
               terminator == token \in {"CR", "LF"}
               boundary == ~ignoredLF /\ terminator /\ ~lineData /\ blockData
           IN /\ complete' = IF boundary THEN Append(complete,pos) ELSE complete
              /\ emitted' = IF boundary THEN Append(emitted,pos) ELSE emitted
              /\ lineData' = IF ignoredLF THEN lineData ELSE ~terminator
              /\ blockData' = IF boundary THEN FALSE ELSE (blockData \/ token = "data")
              /\ skipLF' = (token = "CR")
        /\ pos' = pos + 1
        /\ UNCHANGED <<input,available,done>>
EOF == /\ ~done /\ available = Len(input) /\ pos > available
       /\ emitted' = IF Mode = "legacy" /\ blockData THEN Append(emitted,pos) ELSE emitted
       /\ blockData' = FALSE /\ done' = TRUE
       /\ UNCHANGED <<input,available,pos,lineData,skipLF,complete>>
Next == Deliver \/ Scan \/ EOF \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
OnlyCompleteFrames == emitted = complete
TailDiscarded == done => ~blockData
=======================================================================
