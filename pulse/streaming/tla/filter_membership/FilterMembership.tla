---------------------- MODULE FilterMembership ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode, SameFilter, ExpiringHash, InitialUnknownGroup
VARIABLES record, snapshot, phase, admitted, origin, group, destroyed
vars == <<record, snapshot, phase, admitted, origin, group, destroyed>>
Actors == {1, 2}
Filter(a) == IF SameFilter THEN 1 ELSE a
Empty == [filter |-> 0, members |-> {}]
Init == /\ record = Empty /\ snapshot = [a \in Actors |-> Empty]
        /\ phase = [a \in Actors |-> "new"] /\ admitted = {} /\ origin = 0
        /\ group = InitialUnknownGroup /\ destroyed = FALSE
Read(a) == /\ phase[a] = "new"
           /\ snapshot' = [snapshot EXCEPT ![a] = record]
           /\ phase' = [phase EXCEPT ![a] = "read"]
           /\ UNCHANGED <<record, admitted, origin, group, destroyed>>
Join(a) ==
 /\ phase[a] = "read"
 /\ IF ~ExpiringHash /\ ~(group /\ record.filter = 0) /\ (Mode = "current" \/ snapshot[a].filter \in {0, Filter(a)})
       THEN IF Mode \in {"current", "split"} \/ snapshot[a] = record
              THEN /\ record' = [filter |-> IF snapshot[a].filter = 0 THEN Filter(a) ELSE snapshot[a].filter,
                                  members |-> (IF Mode = "current" THEN record.members ELSE snapshot[a].members) \cup {a}]
                   /\ admitted' = admitted \cup {a}
                   /\ phase' = [phase EXCEPT ![a] = "joined"]
                   /\ origin' = IF origin = 0 THEN Filter(a) ELSE origin
              ELSE /\ phase' = [phase EXCEPT ![a] = "new"]
                   /\ UNCHANGED <<record, admitted, origin>>
       ELSE /\ phase' = [phase EXCEPT ![a] = "rejected"]
            /\ UNCHANGED <<record, admitted, origin>>
 /\ UNCHANGED <<snapshot, group, destroyed>>
Activate(a) == /\ phase[a] = "joined" /\ phase' = [phase EXCEPT ![a] = "active"]
               /\ group' = TRUE
               /\ UNCHANGED <<record, snapshot, admitted, origin, destroyed>>
Leave(a) ==
 /\ phase[a] \in {"joined", "active"}
 /\ admitted' = admitted \ {a}
 /\ phase' = [phase EXCEPT ![a] = "left"]
 /\ record' = IF Mode = "forget" /\ record.members = {a}
                 THEN Empty ELSE [record EXCEPT !.members = @ \ {a}]
 /\ UNCHANGED <<snapshot, origin, group, destroyed>>
RemoveGroup == /\ group /\ record.filter # 0 /\ record.members = {} /\ group' = FALSE
               /\ UNCHANGED <<record, snapshot, phase, admitted, origin, destroyed>>
CoordinatedDestroy == /\ ~destroyed /\ admitted = {} /\ ~group
                      /\ \A a \in Actors: phase[a] # "read"
                      /\ record' = Empty /\ origin' = 0 /\ destroyed' = TRUE
                      /\ UNCHANGED <<snapshot, phase, admitted, group>>
ExpireIdentity == /\ Mode = "ttl" /\ record.filter # 0
                  /\ record' = Empty
                  /\ UNCHANGED <<snapshot, phase, admitted, origin, group, destroyed>>
Next == RemoveGroup \/ CoordinatedDestroy \/ ExpireIdentity
        \/ (\E a \in Actors: Read(a) \/ Join(a) \/ Activate(a) \/ Leave(a))
Spec == Init /\ [][Next]_vars
TypeOK == /\ record.filter \in 0..2 /\ record.members \subseteq Actors
          /\ admitted \subseteq Actors /\ origin \in 0..2
          /\ group \in BOOLEAN /\ destroyed \in BOOLEAN
MembersPreserved == admitted = record.members
Homogeneous == \A a \in admitted: record.filter = Filter(a)
IdentityRetained == origin # 0 => record.filter = origin
MembershipBeforeActivation == \A a \in Actors: phase[a] = "active" => a \in record.members
OnlyPersistentAdmission == ExpiringHash => admitted = {}
UnknownGroupNotClaimed == InitialUnknownGroup /\ ~destroyed => admitted = {}
=============================================================================
