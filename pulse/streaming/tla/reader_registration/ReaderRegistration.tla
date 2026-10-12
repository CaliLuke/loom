----------------------- MODULE ReaderRegistration -----------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES installed, generation, cursor, phase, admitted, attached,
          delivered, subscriber, closing, stale, changedAfterClose
vars == <<installed, generation, cursor, phase, admitted, attached,
          delivered, subscriber, closing, stale, changedAfterClose>>
Init == /\ installed = 1 /\ generation = 1 /\ cursor = 0
        /\ phase = "idle" /\ admitted = 0 /\ attached = 0
        /\ delivered = <<>> /\ subscriber = TRUE /\ closing = FALSE
        /\ stale = FALSE /\ changedAfterClose = FALSE
Admit == /\ phase = "idle" /\ installed # 0 /\ cursor = 0 /\ ~closing
         /\ admitted' = installed /\ phase' = "reading"
         /\ UNCHANGED <<installed, generation, cursor, attached, delivered,
                         subscriber, closing, stale, changedAfterClose>>
Reply == /\ phase = "reading" /\ phase' = "fanout"
         /\ attached' = IF Mode = "reply-identity" THEN installed ELSE admitted
         /\ UNCHANGED <<installed, generation, cursor, admitted, delivered,
                         subscriber, closing, stale, changedAfterClose>>
Deliver == /\ phase = "fanout" /\ ~closing /\ subscriber /\ Len(delivered) < 2
           /\ delivered' = Append(delivered, Len(delivered) + 1)
           /\ UNCHANGED <<installed, generation, cursor, phase, admitted,
                           attached, subscriber, closing, stale, changedAfterClose>>
Complete == /\ phase = "fanout"
            /\ (Len(delivered) = 2 \/ ~subscriber \/ closing)
            /\ phase' = "done"
            /\ LET owns == (installed # 0 /\
                            (Mode = "key-only" \/ installed = attached))
                   update == (owns /\ (~closing \/ Mode = "unchecked-close"))
               IN /\ cursor' = IF update THEN 2 ELSE cursor
                  /\ stale' = (stale \/ (update /\ installed # admitted))
                  /\ changedAfterClose' = (changedAfterClose \/ (update /\ closing))
            /\ UNCHANGED <<installed, generation, admitted, attached,
                            delivered, subscriber, closing>>
Remove == /\ installed # 0 /\ ~closing /\ installed' = 0
          /\ UNCHANGED <<generation, cursor, phase, admitted, attached,
                          delivered, subscriber, closing, stale, changedAfterClose>>
ReAdd == /\ installed = 0 /\ generation < 3 /\ ~closing
         /\ generation' = generation + 1 /\ installed' = generation + 1
         /\ cursor' = 10
         /\ UNCHANGED <<phase, admitted, attached, delivered, subscriber,
                         closing, stale, changedAfterClose>>
Unsubscribe == /\ subscriber /\ subscriber' = FALSE
               /\ UNCHANGED <<installed, generation, cursor, phase, admitted,
                               attached, delivered, closing, stale, changedAfterClose>>
Close == /\ ~closing /\ closing' = TRUE
         /\ UNCHANGED <<installed, generation, cursor, phase, admitted, attached,
                         delivered, subscriber, stale, changedAfterClose>>
Next == Admit \/ Reply \/ Deliver \/ Complete \/ Remove \/ ReAdd \/ Unsubscribe \/ Close
Spec == Init /\ [][Next]_vars
TypeOK == /\ installed \in 0..3 /\ generation \in 1..3 /\ cursor \in {0, 2, 10}
          /\ phase \in {"idle", "reading", "fanout", "done"}
          /\ admitted \in 0..3 /\ attached \in 0..3
          /\ subscriber \in BOOLEAN /\ closing \in BOOLEAN
          /\ stale \in BOOLEAN /\ changedAfterClose \in BOOLEAN
NoStaleAdvance == ~stale
NoPostCloseAdvance == ~changedAfterClose
DeliveryOrder == delivered \in {<<>>, <<1>>, <<1, 2>>}
ReplacementBoundary == installed > 1 => cursor = 10
=============================================================================
