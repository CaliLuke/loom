---------------------------- MODULE JoinRebalance ----------------------------
EXTENDS TLC
CONSTANT Strategy
VARIABLES member, heartbeat, location, acked, due
vars == <<member, heartbeat, location, acked, due>>

\* One start was routed to the old worker before the second worker joined.
\* Membership, heartbeat and that start can reach the local node in any order.
Init == /\ member = FALSE /\ heartbeat = FALSE
        /\ location = "delivery" /\ acked = FALSE /\ due = FALSE
Membership == /\ ~member /\ member' = TRUE /\ due' = TRUE
              /\ UNCHANGED <<heartbeat, location, acked>>
Heartbeat == /\ ~heartbeat /\ heartbeat' = TRUE
             /\ due' = IF Strategy = "heartbeat" THEN TRUE ELSE due
             /\ UNCHANGED <<member, location, acked>>
Start == /\ location = "delivery" /\ location' = "old"
         /\ UNCHANGED <<member, heartbeat, acked, due>>
Ack == /\ location = "old" /\ ~acked /\ acked' = TRUE
       /\ UNCHANGED <<member, heartbeat, location, due>>
Rebalance == /\ due
             /\ LET misplaced == member /\ heartbeat /\ location = "old"
                IN /\ location' = IF misplaced /\ acked THEN "queued" ELSE location
                   \* Existing guard retries cover an unacked start, but not a
                   \* pass that cannot yet see the joining worker or the job.
                   /\ due' = (misplaced /\ ~acked)
             /\ UNCHANGED <<member, heartbeat, acked>>
Deliver == /\ location = "queued" /\ location' = "new"
           /\ UNCHANGED <<member, heartbeat, acked, due>>
Poll == /\ Strategy = "periodic" /\ ~due /\ due' = TRUE
        /\ UNCHANGED <<member, heartbeat, location, acked>>
Next == Membership \/ Heartbeat \/ Start \/ Ack \/ Rebalance \/ Deliver \/ Poll
Spec == Init /\ [][Next]_vars
        /\ WF_vars(Membership) /\ WF_vars(Heartbeat) /\ WF_vars(Start)
        /\ WF_vars(Ack) /\ WF_vars(Rebalance) /\ WF_vars(Deliver) /\ WF_vars(Poll)
OwnerReached == <>[](location = "new")
NoPrematureMove == location \in {"queued", "new"} => member /\ heartbeat /\ acked
=============================================================================
