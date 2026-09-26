--------------------------- MODULE SinkAddStream ---------------------------
(***************************************************************************)
(* How instances of one sink add and remove a stream (issue #481).        *)
(*                                                                         *)
(* Every in-process instance of a sink has its own consumer. The          *)
(* replicated consumers map of a stream holds, under the sink name, the   *)
(* consumers of the instances that added the stream. Sink.AddStream adds  *)
(* the consumer to that map and creates the consumer group, keeping an    *)
(* existing one. Sink.RemoveStream removes the consumer from the map and  *)
(* destroys the group when no consumer remains. Each map update and each  *)
(* group command is one atomic Redis step; the steps of one call are not  *)
(* atomic together, so the calls of two instances interleave.             *)
(*                                                                         *)
(* Toggles:                                                                *)
(*   CREATE_FIRST   AddStream creates the group before it adds the        *)
(*                  consumer to the map. FALSE is the order of main.      *)
(*   ROLLBACK       AddStream removes the consumer from the map again     *)
(*                  when the group creation fails. FALSE is main before   *)
(*                  the fix of #481.                                       *)
(*   REMOVE         How RemoveStream destroys the group when the map      *)
(*                  update leaves no consumer (issue #508):               *)
(*                  "split"   a separate XGROUP DESTROY: main before the  *)
(*                            fix of #508.                                 *)
(*                  "atomic"  the map update and the destroy are one      *)
(*                            step: a candidate fix.                       *)
(*                  "checked" a separate step that destroys the group     *)
(*                            only if the map still holds no consumer:    *)
(*                            the fix of #508 (destroyGroupScript in      *)
(*                            sink.go).                                    *)
(***************************************************************************)
EXTENDS FiniteSets

CONSTANTS Instances, CREATE_FIRST, ROLLBACK, REMOVE

ASSUME REMOVE \in {"split", "atomic", "checked"}

VARIABLES
    members,  \* consumers the map holds under the sink name
    group,    \* whether the consumer group exists
    pc        \* the step each instance is at

vars == <<members, group, pc>>

(* "out": the stream is not part of the sink; "in": AddStream returned nil.
   The other values are the steps inside AddStream ("add_*") and
   RemoveStream ("rm_*"). *)
Steps == {"out", "add_map", "add_group", "add_rollback", "in", "rm_destroy"}

TypeOK ==
    /\ members \subseteq Instances
    /\ group \in BOOLEAN
    /\ pc \in [Instances -> Steps]

Init ==
    /\ members = {}
    /\ group = FALSE
    /\ pc = [i \in Instances |-> "out"]

\* AddStream starts with the map update or with the group creation.
StartAdd(i) ==
    /\ pc[i] = "out"
    /\ pc' = [pc EXCEPT ![i] = IF CREATE_FIRST THEN "add_group" ELSE "add_map"]
    /\ UNCHANGED <<members, group>>

AddToMap(i) ==
    /\ pc[i] = "add_map"
    /\ members' = members \cup {i}
    /\ pc' = [pc EXCEPT ![i] = IF CREATE_FIRST THEN "in" ELSE "add_group"]
    /\ UNCHANGED group

\* XGROUP CREATE MKSTREAM; BUSYGROUP counts as success.
CreateGroupOk(i) ==
    /\ pc[i] = "add_group"
    /\ group' = TRUE
    /\ pc' = [pc EXCEPT ![i] = IF CREATE_FIRST THEN "add_map" ELSE "in"]
    /\ UNCHANGED members

\* Any other error: AddStream returns it.
CreateGroupFails(i) ==
    /\ pc[i] = "add_group"
    /\ pc' = [pc EXCEPT ![i] =
                IF ~CREATE_FIRST /\ ROLLBACK THEN "add_rollback" ELSE "out"]
    /\ UNCHANGED <<members, group>>

Rollback(i) ==
    /\ pc[i] = "add_rollback"
    /\ members' = members \ {i}
    /\ pc' = [pc EXCEPT ![i] = "out"]
    /\ UNCHANGED group

\* rmap RemoveValues, which returns the consumers that remain.
RemoveFromMap(i) ==
    LET remains == members \ {i} IN
    /\ pc[i] = "in"
    /\ members' = remains
    /\ IF REMOVE = "atomic"
          THEN /\ group' = (group /\ remains # {})
               /\ pc' = [pc EXCEPT ![i] = "out"]
          ELSE /\ pc' = [pc EXCEPT ![i] = IF remains = {} THEN "rm_destroy" ELSE "out"]
               /\ UNCHANGED group

\* "split": XGROUP DESTROY. "checked": destroyGroupScript, which destroys the
\* group only if the map holds no consumer under the sink name, in the same
\* script.
DestroyGroup(i) ==
    /\ pc[i] = "rm_destroy"
    /\ group' = IF REMOVE = "checked" /\ members # {} THEN group ELSE FALSE
    /\ pc' = [pc EXCEPT ![i] = "out"]
    /\ UNCHANGED members

Next ==
    \E i \in Instances :
        \/ StartAdd(i)
        \/ AddToMap(i)
        \/ CreateGroupOk(i)
        \/ CreateGroupFails(i)
        \/ Rollback(i)
        \/ RemoveFromMap(i)
        \/ DestroyGroup(i)

Spec == Init /\ [][Next]_vars

(* The map holds no consumer of an instance whose stream is not added:
   issue #481. *)
NoStaleMember == \A i \in Instances : pc[i] = "out" => i \notin members

(* An instance whose AddStream succeeded reads from an existing group. *)
AddedHasGroup == \A i \in Instances : pc[i] = "in" => group

=============================================================================
