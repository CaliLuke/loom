------------------------- MODULE HeartbeatRetirement ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES members, native, fresh, pending, groupAlive, staleSnapshot,
          checkedFresh, checkedVersion, version, phase, candidate
vars == <<members, native, fresh, pending, groupAlive, staleSnapshot,
         checkedFresh, checkedVersion, version, phase, candidate>>

Init == /\ members = {"closed", "active"}
        /\ native = members
        /\ fresh = {}
        /\ pending = {}
        /\ groupAlive = TRUE
        /\ staleSnapshot = {}
        /\ checkedFresh = {}
        /\ checkedVersion = 0
        /\ version = 0
        /\ phase = "idle"
        /\ candidate = ""

Observe == /\ phase = "idle"
           /\ staleSnapshot' = members \ fresh
           /\ phase' = "observed"
           /\ UNCHANGED <<members, native, fresh, pending, groupAlive, checkedFresh,
                           checkedVersion, version, candidate>>

DeleteNative(a) == /\ phase = "observed"
                  /\ a \in staleSnapshot
                  /\ native' = native \ {a}
                  /\ candidate' = a
                  /\ phase' = "native_deleted"
                  /\ UNCHANGED <<members, fresh, pending, groupAlive,
                                  staleSnapshot, checkedFresh,
                                  checkedVersion, version>>

Refresh == /\ "active" \in members
          /\ "active" \notin fresh
          /\ fresh' = fresh \cup {"active"}
          /\ version' = version + 1
          /\ UNCHANGED <<members, native, pending, groupAlive,
                          staleSnapshot, checkedFresh,
                          checkedVersion, phase, candidate>>

RecheckMembership == /\ phase = "native_deleted"
                    /\ checkedFresh' = fresh
                    /\ checkedVersion' = version
                    /\ phase' = "checked"
                    /\ UNCHANGED <<members, native, fresh, pending, groupAlive,
                                    staleSnapshot,
                                    version, candidate>>

DeliverPending == /\ phase = "native_deleted"
                  /\ groupAlive
                  /\ pending = {}
                  /\ pending' = {candidate}
                  /\ UNCHANGED <<members, native, fresh, groupAlive,
                                  staleSnapshot, checkedFresh, checkedVersion,
                                  version, phase, candidate>>

AcknowledgePending == /\ pending # {}
                     /\ pending' = {}
                     /\ UNCHANGED <<members, native, fresh, groupAlive,
                                     staleSnapshot, checkedFresh, checkedVersion,
                                     version, phase, candidate>>

RetirementAllowed == IF Mode = "snapshot"
                     THEN candidate \in staleSnapshot
                     ELSE /\ phase = "checked"
                          /\ candidate \notin checkedFresh
                          /\ (Mode \notin {"versioned-cas", "atomic-pel-guard"}
                              \/ checkedVersion = version)

RetireMembership == /\ phase \in {"native_deleted", "checked"}
                   /\ candidate \in members
                   /\ RetirementAllowed
                   /\ members' = members \ {candidate}
                   /\ phase' = "idle"
                   /\ staleSnapshot' = {}
                   /\ candidate' = ""
                   /\ UNCHANGED <<native, fresh, pending, groupAlive, checkedFresh,
                                   checkedVersion, version>>

DestroyEmptyGroup == /\ phase = "idle"
                     /\ members = {}
                     /\ groupAlive
                     /\ (Mode # "atomic-pel-guard" \/ pending = {})
                     /\ groupAlive' = FALSE
                     /\ UNCHANGED <<members, native, fresh, pending, staleSnapshot,
                                     checkedFresh, checkedVersion, version,
                                     phase, candidate>>

Next == Observe \/ Refresh \/ RecheckMembership \/ DeliverPending \/
        AcknowledgePending \/ DestroyEmptyGroup \/
        (\E a \in members: DeleteNative(a)) \/ RetireMembership
Spec == Init /\ [][Next]_vars
FreshPreserved == fresh \subseteq members
PendingPreserved == pending = {} \/ groupAlive
=============================================================================
