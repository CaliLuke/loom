---------------------- MODULE MembershipRetirement ----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES members, native, fresh, pending
vars == <<members, native, fresh, pending>>
Actors == {"closed", "joining", "pending"}
Init == /\ members = Actors
        /\ native = {"pending"}
        /\ fresh = {"joining"}
        /\ pending = {"pending"}
Retire(a) ==
 /\ a \in members /\ a \notin pending
 /\ CASE Mode = "native_only" -> a \in native /\ a \notin fresh
         [] Mode = "absent_only" -> a \notin native
         [] OTHER -> a \notin fresh
 /\ members' = members \ {a}
 /\ native' = native \ {a}
 /\ UNCHANGED <<fresh, pending>>
Activate == /\ "joining" \in members /\ "joining" \notin native
            /\ native' = native \cup {"joining"}
            /\ UNCHANGED <<members, fresh, pending>>
Next == Activate \/ (\E a \in Actors: Retire(a))
Spec == Init /\ [][Next]_vars /\ \A a \in Actors: WF_vars(Retire(a))
FreshPreserved == fresh \subseteq members
PendingPreserved == pending \subseteq native /\ pending \subseteq members
ClosedRetired == <> ("closed" \notin members)
=============================================================================
