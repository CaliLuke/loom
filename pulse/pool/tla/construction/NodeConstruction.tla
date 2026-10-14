----------------------- MODULE NodeConstruction -----------------------
EXTENDS Naturals, Sequences, FiniteSets

CONSTANT Mode
VARIABLE s
vars == <<s>>

Steps == <<"registration", "shutdown", "heartbeatMap", "otherMaps",
           "sink", "privateStream", "reader", "cancel">>
Remote == {"protocol", "heartbeat", "privateStream"}
Writes == {"registration", "privateStream"}
Obligations(item) == IF item = "registration" THEN <<"heartbeat", "protocol">>
                     ELSE <<item>>
Local == {"shutdown", "heartbeatMap", "otherMaps", "sink", "reader", "cancel"}
Elements(xs) == {xs[i] : i \in 1..Len(xs)}
Reverse(xs) == [i \in 1..Len(xs) |-> xs[Len(xs) - i + 1]]

Init == s = [phase |-> "acquire", next |-> 1, stack |-> <<>>,
             locals |-> {}, runtime |-> {}, attempted |-> {}, remote |-> {},
             acquired |-> <<>>, released |-> <<>>, withdrawn |-> {},
             original |-> FALSE, cleanupErrors |-> {}, returned |-> {},
             transfers |-> 0, shared |-> "preserved"]

\* One grouped otherMaps slot represents seven independent identically owned
\* handles; ordering within that slot is checked by the Go owner test.
AcquireLocal ==
  /\ s.phase = "acquire" /\ s.next <= Len(Steps)
  /\ LET item == Steps[s.next] IN
     /\ item \in Local
     /\ s' = [s EXCEPT !.next = @ + 1,
                       !.locals = @ \cup {item},
                       !.stack = Append(@, item)]

\* Admission and XADD may commit although their reply reports failure. Enroll
\* intent before execution, not only after a successful reply.
AttemptRemote ==
  /\ s.phase = "acquire" /\ s.next <= Len(Steps)
  /\ LET item == Steps[s.next] IN
     /\ item \in Writes
     /\ \E replyOK \in BOOLEAN, committed \in BOOLEAN :
        /\ replyOK => committed
        /\ s' = [s EXCEPT
             !.next = @ + 1,
             !.attempted = @ \cup Elements(Obligations(item)),
             !.remote = IF committed THEN @ \cup Elements(Obligations(item)) ELSE @,
             !.stack = IF Mode = "successful_only" /\ ~replyOK
                       THEN @ ELSE @ \o Obligations(item),
             !.acquired = IF replyOK THEN @ ELSE
                           IF Mode = "successful_only" THEN s.stack
                           ELSE s.stack \o Obligations(item),
             !.phase = IF replyOK THEN "acquire"
                       ELSE IF Mode = "current" THEN "failed" ELSE "abort",
             !.original = ~replyOK,
             !.returned = IF ~replyOK /\ Mode = "current" THEN {"original"} ELSE @]

\* Early validation rejection or a failed local acquisition owns no new handle.
Fail ==
  /\ s.phase = "acquire"
  /\ s' = [s EXCEPT !.phase = IF Mode = "current" THEN "failed" ELSE "abort",
                    !.original = TRUE, !.acquired = s.stack,
                    !.returned = IF Mode = "current" THEN {"original"} ELSE @]

Transfer ==
  /\ s.phase = "acquire" /\ s.next = Len(Steps) + 1
  /\ s' = [s EXCEPT !.phase = "running", !.runtime = s.locals,
                    !.locals = {}, !.stack = <<>>, !.transfers = @ + 1]

ReleaseLocal ==
  /\ s.phase = "abort" /\ Len(s.stack) > 0
  /\ LET item == s.stack[Len(s.stack)] IN
     /\ item \in Local
     /\ s' = [s EXCEPT !.locals = @ \ {item},
                       !.stack = SubSeq(@, 1, Len(@) - 1),
                       !.released = Append(@, item)]

\* A failed cleanup may have deleted its own remote resource, or none. It must
\* not prevent subsequent local releases or erase the original cause.
WithdrawRemote ==
  /\ s.phase = "abort" /\ Len(s.stack) > 0
  /\ LET item == s.stack[Len(s.stack)] IN
     /\ item \in Remote
     /\ \E cleanupOK \in BOOLEAN, deleted \in BOOLEAN :
        /\ cleanupOK => deleted
        /\ s' = [s EXCEPT
             !.remote = IF deleted THEN @ \ {item} ELSE @,
             !.withdrawn = @ \cup {item},
             !.stack = SubSeq(@, 1, Len(@) - 1),
             !.released = Append(@, item),
             !.cleanupErrors = IF cleanupOK THEN @ ELSE @ \cup {item},
             !.phase = IF ~cleanupOK /\ Mode = "stop_on_error" THEN "failed" ELSE @,
             !.returned = IF ~cleanupOK /\ Mode = "stop_on_error"
                          THEN {"original"} \cup s.cleanupErrors \cup {item} ELSE @]

FinishAbort ==
  /\ s.phase = "abort" /\ s.stack = <<>>
  /\ s' = [s EXCEPT !.phase = "failed",
                    !.returned = {"original"} \cup s.cleanupErrors]

Next == AcquireLocal \/ AttemptRemote \/ Fail \/ Transfer
        \/ ReleaseLocal \/ WithdrawRemote \/ FinishAbort
Spec == Init /\ [][Next]_vars

LocalRelease == s.phase = "failed" => s.locals = {}
AttemptedWithdrawal == s.phase = "failed" => s.attempted \subseteq s.withdrawn
ReverseRelease == s.released = SubSeq(Reverse(s.acquired), 1, Len(s.released))
ErrorsPreserved == s.phase = "failed" =>
                     s.original /\ s.returned = {"original"} \cup s.cleanupErrors
SingleOwner == s.locals \cap s.runtime = {} /\ s.transfers <= 1
               /\ (s.phase = "running" => s.transfers = 1 /\ s.stack = <<>>)
NoSharedDeletion == s.shared = "preserved"
NoFailedRuntime == s.phase = "failed" => s.runtime = {}
=============================================================================
