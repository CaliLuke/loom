--------------------------- MODULE SinkAddStream ---------------------------
(***************************************************************************)
(* How instances of one sink add and remove a stream (issues #481, #508   *)
(* and #531).                                                              *)
(*                                                                         *)
(* Every in-process instance of a sink has its own consumer. The          *)
(* replicated consumers map of a stream holds, under the sink name, the   *)
(* consumers of the instances that added the stream. NewSink and          *)
(* Sink.AddStream add the consumer to that map and create the consumer    *)
(* group, keeping an existing one. NewSink also creates the consumer      *)
(* (XGROUP CREATECONSUMER), which fails when the group is missing         *)
(* (NOGROUP). Sink.RemoveStream removes the consumer from the map and     *)
(* destroys the group when no consumer remains. Each map update and each  *)
(* group command is one atomic Redis step; the steps of one call are not  *)
(* atomic together, so the calls of two instances interleave.             *)
(*                                                                         *)
(* Toggles:                                                                *)
(*   CREATE_FIRST   AddStream creates the group before it adds the        *)
(*                  consumer to the map. FALSE is the order of main.      *)
(*   ROLLBACK       AddStream, and NewSink with NEW_SINK = "map_first",   *)
(*                  remove the consumer from the map again when a later   *)
(*                  step fails. FALSE is main before the fix of #481.     *)
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
(*   NEW_SINK       How NewSink orders its steps (issue #531):             *)
(*                  "none"          no instance calls NewSink: the model  *)
(*                                  of AddStream and RemoveStream only.   *)
(*                  "create_first"  the group, the consumer, then the     *)
(*                                  map: main before the fix of #531.     *)
(*                  "map_first"     the map, the group, then the          *)
(*                                  consumer, as AddStream: the fix of    *)
(*                                  #531.                                  *)
(***************************************************************************)
EXTENDS FiniteSets

CONSTANTS Instances, CREATE_FIRST, ROLLBACK, REMOVE, NEW_SINK

ASSUME REMOVE \in {"split", "atomic", "checked"}
ASSUME NEW_SINK \in {"none", "create_first", "map_first"}

VARIABLES
    members,  \* consumers the map holds under the sink name
    group,    \* whether the consumer group exists
    pc        \* the step each instance is at

vars == <<members, group, pc>>

(* "out": the stream is not part of the sink; "in": AddStream or NewSink
   returned nil. The other values are the steps inside AddStream ("add_*"),
   NewSink ("new_*") and RemoveStream ("rm_*"). *)
Steps == {"out", "add_map", "add_group", "add_rollback", "in", "rm_destroy",
          "new_map", "new_group", "new_consumer", "new_rollback"}

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

(* NewSink creates a sink instance that reads the stream. An instance that
   removed the stream and calls NewSink again stands for a new instance of
   the sink. *)
StartNew(i) ==
    /\ NEW_SINK # "none"
    /\ pc[i] = "out"
    /\ pc' = [pc EXCEPT ![i] = IF NEW_SINK = "map_first" THEN "new_map" ELSE "new_group"]
    /\ UNCHANGED <<members, group>>

NewAddToMap(i) ==
    /\ pc[i] = "new_map"
    /\ members' = members \cup {i}
    /\ pc' = [pc EXCEPT ![i] = IF NEW_SINK = "map_first" THEN "new_group" ELSE "in"]
    /\ UNCHANGED group

\* The step after a failed group or consumer creation of NewSink: the
\* rollback when the consumer is in the map, the end otherwise.
NewFailed == IF NEW_SINK = "map_first" /\ ROLLBACK THEN "new_rollback" ELSE "out"

\* XGROUP CREATE MKSTREAM; BUSYGROUP counts as success.
NewCreateGroupOk(i) ==
    /\ pc[i] = "new_group"
    /\ group' = TRUE
    /\ pc' = [pc EXCEPT ![i] = "new_consumer"]
    /\ UNCHANGED members

\* Any other error: NewSink returns it.
NewCreateGroupFails(i) ==
    /\ pc[i] = "new_group"
    /\ pc' = [pc EXCEPT ![i] = NewFailed]
    /\ UNCHANGED <<members, group>>

\* XGROUP CREATECONSUMER and the keep-alive update succeed. XGROUP
\* CREATECONSUMER needs the group.
NewCreateConsumerOk(i) ==
    /\ pc[i] = "new_consumer"
    /\ group
    /\ pc' = [pc EXCEPT ![i] = IF NEW_SINK = "map_first" THEN "in" ELSE "new_map"]
    /\ UNCHANGED <<members, group>>

\* NOGROUP or any other error of either command: NewSink returns it.
NewCreateConsumerFails(i) ==
    /\ pc[i] = "new_consumer"
    /\ pc' = [pc EXCEPT ![i] = NewFailed]
    /\ UNCHANGED <<members, group>>

\* As the rollback of AddStream, it keeps the group.
NewRollback(i) ==
    /\ pc[i] = "new_rollback"
    /\ members' = members \ {i}
    /\ pc' = [pc EXCEPT ![i] = "out"]
    /\ UNCHANGED group

Next ==
    \E i \in Instances :
        \/ StartAdd(i)
        \/ AddToMap(i)
        \/ CreateGroupOk(i)
        \/ CreateGroupFails(i)
        \/ Rollback(i)
        \/ RemoveFromMap(i)
        \/ DestroyGroup(i)
        \/ StartNew(i)
        \/ NewAddToMap(i)
        \/ NewCreateGroupOk(i)
        \/ NewCreateGroupFails(i)
        \/ NewCreateConsumerOk(i)
        \/ NewCreateConsumerFails(i)
        \/ NewRollback(i)

Spec == Init /\ [][Next]_vars

(* The map holds no consumer of an instance whose stream is not added:
   issue #481. *)
NoStaleMember == \A i \in Instances : pc[i] = "out" => i \notin members

(* An instance whose AddStream or NewSink succeeded reads from an existing
   group. *)
AddedHasGroup == \A i \in Instances : pc[i] = "in" => group

=============================================================================
