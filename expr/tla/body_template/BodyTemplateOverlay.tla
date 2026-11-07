------------------------ MODULE BodyTemplateOverlay ------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Behavior
VARIABLES phase, explicitType, authoredDefault, effectiveDefault, stage,
          requested, executed, callbacks, contexts, types, defaults, fields,
          pointers, heap, changed, compositionIdentity
vars == <<phase, explicitType, authoredDefault, effectiveDefault, stage,
          requested, executed, callbacks, contexts, types, defaults, fields,
          pointers, heap, changed, compositionIdentity>>
Uses == {1, 2}
Members == {"count", "groups", "name"}
Selected(u) == IF u = 1 THEN {"count", "groups"} ELSE {"count"}
EffectiveTypes == [m \in Members |-> CASE m = "count" -> "Int"
                                      [] m = "groups" -> "Array"
                                      [] OTHER -> "String"]
RawTypes == [m \in Members |-> IF m = "groups" THEN "Object" ELSE "String"]
EarlyTypes == [EffectiveTypes EXCEPT !["groups"] = "Object"]

Init == /\ phase = "dsl"
        /\ explicitType \in {"Int", "String"}
        /\ authoredDefault \in IF explicitType = "Int" THEN {7, 9} ELSE {"authored"}
        /\ effectiveDefault \in {7, 8}
        /\ stage = [u \in Uses |-> "registered"]
        /\ requested = {}
        /\ executed = {}
        /\ callbacks = [u \in Uses |-> 0]
        /\ contexts = [u \in Uses |-> 0]
        /\ types = [u \in Uses |-> RawTypes]
        /\ defaults = [u \in Uses |-> 0]
        /\ fields = [u \in Uses |-> {}]
        /\ pointers = [u \in Uses |-> u]
        /\ heap = [i \in 0..2 |-> "original"]
        /\ changed = {}
        /\ compositionIdentity = [u \in Uses |-> FALSE]

\* Consumer 2 creates a generated CollectionOf declaration in its callback.
\* Current DSL scheduling executes that generated root after body callbacks.
Author(u) ==
    /\ stage[u] = "registered"
    /\ phase = IF Behavior = "defer-callback" THEN "prepare" ELSE "dsl"
    /\ requested' = IF u = 2 THEN requested \cup {u} ELSE requested
    /\ callbacks' = [callbacks EXCEPT ![u] = @ + 1]
    /\ contexts' = [contexts EXCEPT ![u] = u]
    /\ stage' = [stage EXCEPT ![u] = "authored"]
    /\ types' = [types EXCEPT ![u] =
         IF Behavior = "current" THEN RawTypes ELSE EarlyTypes]
    /\ defaults' = [defaults EXCEPT ![u] = IF Behavior = "current" THEN 0 ELSE 7]
    /\ UNCHANGED <<phase, explicitType, authoredDefault, effectiveDefault,
                   executed, fields, pointers, heap, changed, compositionIdentity>>

FinishAllDSL ==
    /\ phase = "dsl"
    /\ (Behavior = "defer-callback" \/ (\A u \in Uses: stage[u] = "authored"))
    /\ phase' = "prepare"
    /\ executed' = requested
    /\ UNCHANGED <<explicitType, authoredDefault, effectiveDefault, stage,
                   requested, callbacks, contexts, types, defaults, fields,
                   pointers, heap, changed, compositionIdentity>>

Publish(u) ==
    /\ phase = "prepare"
    /\ stage[u] = "authored"
    /\ LET template == IF Behavior \in {"current", "eager"}
                       THEN types[u] ELSE EffectiveTypes
           inherited == IF Behavior \in {"current", "eager"}
                        THEN defaults[u] ELSE effectiveDefault
           keepLocal == u = 2 /\ Behavior # "overwrite" /\
                        ~(Behavior = "diff-overlay" /\ authoredDefault = 7)
       IN /\ types' = [types EXCEPT ![u] =
                IF u = 2 /\ Behavior # "overwrite"
                THEN [template EXCEPT !["count"] = explicitType] ELSE template]
          /\ defaults' = [defaults EXCEPT ![u] = IF keepLocal THEN authoredDefault ELSE inherited]
    /\ fields' = [fields EXCEPT ![u] = IF Behavior = "whole-source" THEN Members ELSE Selected(u)]
    /\ pointers' = [pointers EXCEPT ![u] = IF Behavior = "shared" THEN 0 ELSE u]
    /\ compositionIdentity' = [compositionIdentity EXCEPT ![u] = Behavior = "whole-source"]
    /\ stage' = [stage EXCEPT ![u] = "published"]
    /\ UNCHANGED <<phase, explicitType, authoredDefault, effectiveDefault,
                   requested, executed, callbacks, contexts, heap, changed>>

Mutate(u) ==
    /\ stage[u] = "published"
    /\ u \notin changed
    /\ heap' = [heap EXCEPT ![pointers[u]] = "changed"]
    /\ changed' = changed \cup {u}
    /\ UNCHANGED <<phase, explicitType, authoredDefault, effectiveDefault, stage,
                   requested, executed, callbacks, contexts, types, defaults,
                   fields, pointers, compositionIdentity>>

Next == FinishAllDSL \/ (\E u \in Uses: Author(u) \/ Publish(u) \/ Mutate(u))
Spec == Init /\ [][Next]_vars
AllAuthoredDeclarationsExecuted == \A u \in Uses:
    stage[u] = "published" => requested \subseteq executed
SelectedOnly == \A u \in Uses: stage[u] = "published" => fields[u] = Selected(u)
InheritedComplete == stage[1] = "published" =>
    /\ types[1]["count"] = "Int"
    /\ types[1]["groups"] = "Array"
    /\ defaults[1] = effectiveDefault
AuthoredChoiceRetained == stage[2] = "published" =>
    /\ types[2]["count"] = explicitType
    /\ defaults[2] = authoredDefault
CallbackContract == \A u \in Uses:
    /\ callbacks[u] <= 1
    /\ (stage[u] \in {"authored", "published"} => callbacks[u] = 1 /\ contexts[u] = u)
NoSourceIdentityOnSubset == \A u \in Uses: ~compositionIdentity[u]
SourceUnchanged == heap[0] = "original"
PeerUnchanged == \A u \in Uses:
    (stage[u] = "published" /\ u \notin changed) => heap[pointers[u]] = "original"
=============================================================================
