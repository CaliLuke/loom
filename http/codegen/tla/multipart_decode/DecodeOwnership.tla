------------------------ MODULE DecodeOwnership ------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES custom, elements, cookie, bodyOK, elementsOK, basicOK,
          phase, declared, arityOK, rejected, invoked
vars == <<custom, elements, cookie, bodyOK, elementsOK, basicOK,
          phase, declared, arityOK, rejected, invoked>>
Init == /\ custom \in BOOLEAN
        /\ elements \in BOOLEAN
        /\ cookie \in BOOLEAN
        /\ cookie => elements
        /\ bodyOK \in BOOLEAN
        /\ elementsOK \in BOOLEAN
        /\ basicOK \in BOOLEAN
        /\ ~elements => elementsOK
        /\ phase = "emit"
        /\ declared = TRUE /\ arityOK = TRUE
        /\ rejected = FALSE /\ invoked = FALSE
Emit == /\ phase = "emit"
        /\ declared' = ~(Mode = "legacy" /\ custom /\ elements)
        /\ arityOK' = ~(Mode \in {"legacy", "declaration-only"} /\ custom /\ cookie)
        /\ phase' = "body"
        /\ UNCHANGED <<custom, elements, cookie, bodyOK, elementsOK, basicOK, rejected, invoked>>
Body == /\ phase = "body"
        /\ rejected' = ~bodyOK
        /\ phase' = IF bodyOK THEN "elements" ELSE "done"
        /\ UNCHANGED <<custom, elements, cookie, bodyOK, elementsOK, basicOK, declared, arityOK, invoked>>
Elements == /\ phase = "elements"
            /\ rejected' = ~elementsOK
            /\ phase' = IF elementsOK THEN "basic" ELSE "done"
            /\ UNCHANGED <<custom, elements, cookie, bodyOK, elementsOK, basicOK, declared, arityOK, invoked>>
Basic == /\ phase = "basic"
         /\ rejected' = ~basicOK
         /\ invoked' = basicOK
         /\ phase' = "done"
         /\ UNCHANGED <<custom, elements, cookie, bodyOK, elementsOK, basicOK, declared, arityOK>>
Next == Emit \/ Body \/ Elements \/ Basic
Spec == Init /\ [][Next]_vars
Compiles == declared /\ arityOK
ServiceBarrier == invoked => bodyOK /\ elementsOK /\ basicOK /\ ~rejected
Completed == phase = "done" => invoked = (bodyOK /\ elementsOK /\ basicOK)
=============================================================================
