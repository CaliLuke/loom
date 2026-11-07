------------------------- MODULE ObjectHelperPresence -------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Cases == [firstRequired: BOOLEAN, pointerLayout: BOOLEAN,
          hasFields: BOOLEAN, input: {"nil", "empty", "value"}]
VARIABLES current, done, crashed, output
vars == <<current, done, crashed, output>>
Guard(required, pointerLayout) ==
  IF Mode = "old" THEN pointerLayout \/ ~required ELSE TRUE
Init == /\ current \in Cases /\ done = FALSE
        /\ crashed = FALSE /\ output = "nil"
Convert ==
  /\ ~done /\ done' = TRUE /\ UNCHANGED current
  /\ crashed' = (current.input = "nil" /\ current.hasFields /\
                  ~Guard(current.firstRequired, current.pointerLayout))
  /\ output' = IF current.input = "nil" /\
                  ~Guard(current.firstRequired, current.pointerLayout)
               THEN "empty" ELSE current.input
Next == Convert \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
NoPanic == ~crashed
PresencePreserved == done => output = current.input
RequirednessIndependent ==
  Guard(TRUE, current.pointerLayout) = Guard(FALSE, current.pointerLayout)
Terminates == <>done
=============================================================================
