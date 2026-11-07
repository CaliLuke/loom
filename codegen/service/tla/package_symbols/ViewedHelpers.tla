------------------------- MODULE ViewedHelpers -----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT OwnedHelpers
VARIABLES scenario, pendingTypes, pendingHelpers, occupied, types, helpers, references
Scenarios == {"projection", "wrapper", "unwrap", "reconstruct", "cross"}
Base(h) == CASE h = "projection" -> "ProjectFoo"
             [] h \in {"Foo/bar", "FooBar/default"} -> "ProjectFooBar"
             [] h = "wrapper" -> "NewViewedFoo"
             [] h = "unwrap" -> "NewFoo"
             [] OTHER -> "NewFooFromFooView"
Helpers == IF scenario = "cross" THEN {"Foo/bar", "FooBar/default"} ELSE {scenario}
Types == IF scenario = "cross" THEN {"ProjectFooBar2"} ELSE {Base(scenario)}
Candidates(n) == <<n, n \o "2", n \o "3", n \o "4", n \o "5", n \o "6", n \o "7">>
Fresh(n, used) ==
    LET choices == Candidates(n)
        free == {i \in 1..Len(choices) : choices[i] \notin used}
        first == CHOOSE i \in free : \A j \in free : i <= j
    IN choices[first]
vars == <<scenario, pendingTypes, pendingHelpers, occupied, types, helpers, references>>
Init == /\ scenario \in Scenarios
        /\ pendingTypes = Types /\ pendingHelpers = Helpers /\ occupied = {}
        /\ types = [t \in Types |-> ""] /\ helpers = [h \in Helpers |-> ""]
        /\ references = [h \in Helpers |-> ""]
Nominal(t) ==
    LET name == Fresh(t, occupied)
    IN /\ t \in pendingTypes /\ pendingTypes' = pendingTypes \ {t}
       /\ types' = [types EXCEPT ![t] = name] /\ occupied' = occupied \cup {name}
       /\ UNCHANGED <<scenario, pendingHelpers, helpers, references>>
Helper(h) ==
    LET name == IF OwnedHelpers THEN Fresh(Base(h), occupied) ELSE Base(h)
    IN /\ h \in pendingHelpers /\ pendingHelpers' = pendingHelpers \ {h}
       /\ helpers' = [helpers EXCEPT ![h] = name]
       /\ occupied' = IF OwnedHelpers THEN occupied \cup {name} ELSE occupied
       /\ UNCHANGED <<scenario, pendingTypes, types, references>>
Reference(h) == /\ h \in Helpers \ pendingHelpers
                /\ references' = [references EXCEPT ![h] = helpers[h]]
                /\ UNCHANGED <<scenario, pendingTypes, pendingHelpers, occupied, types, helpers>>
Next == (\E t \in pendingTypes : Nominal(t))
        \/ (\E h \in pendingHelpers : Helper(h))
        \/ (\E h \in Helpers \ pendingHelpers : Reference(h))
DeclarationsDistinct ==
    /\ \A a,b \in Helpers \ pendingHelpers : a # b => helpers[a] # helpers[b]
    /\ \A h \in Helpers \ pendingHelpers : \A t \in Types \ pendingTypes : helpers[h] # types[t]
ReferencesStable == \A h \in Helpers : references[h] # "" => references[h] = helpers[h]
=============================================================================
