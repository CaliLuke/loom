-------------------------- MODULE WireBindings --------------------------
EXTENDS Integers, FiniteSets, Sequences
CONSTANT Mode
VARIABLES slot, source, codec, origin, together, accepted, phase
vars == <<slot, source, codec, origin, together, accepted, phase>>
Bindings == 1..2
Sources == {<<1>>, <<2>>, <<3>>, <<1, 2>>, <<2, 1>>, <<1, 3>>, <<3, 1>>, <<2, 3>>, <<3, 2>>}
SourceFields(i) == {source[i][j] : j \in 1..Len(source[i])}
Families == {"basic", "bearer", "plain"}
Domain(i) == IF codec[i] = "raw" THEN Families ELSE {codec[i]}
SameIdentity == source[1] = source[2] /\ codec[1] = codec[2]
SeparateSources == SourceFields(1) \cap SourceFields(2) = {}
SeparateSlots == slot[1] # slot[2]
DisjointFrames == Domain(1) \cap Domain(2) = {}
AlternativeOnly == ~together
Candidate == SeparateSlots \/ SameIdentity \/
             (AlternativeOnly /\ SeparateSources /\ DisjointFrames)
Init == /\ slot \in [Bindings -> 1..2]
        /\ source \in [Bindings -> Sources]
        /\ codec \in [Bindings -> {"raw", "basic", "bearer"}]
        /\ origin \in [Bindings -> {"explicit", "implicit"}]
        /\ \A i \in Bindings : (codec[i] = "basic") <=> (Len(source[i]) = 2)
        /\ ((codec[1] = "basic") /\ (codec[2] = "basic")) => source[1] = source[2]
        /\ together \in BOOLEAN
        /\ accepted = FALSE
        /\ phase = "classify"
Classify == /\ phase = "classify"
            /\ accepted' = IF Mode = "legacy" THEN TRUE
                            ELSE IF Mode = "blanket" THEN SeparateSlots \/ SameIdentity
                            ELSE IF Mode = "explicit-only"
                            THEN origin[1] = "implicit" \/ origin[2] = "implicit" \/ Candidate
                            ELSE Candidate
            /\ phase' = "done"
            /\ UNCHANGED <<slot, source, codec, origin, together>>
Next == Classify
Spec == Init /\ [][Next]_vars
NoConjunctiveOverwrite == phase = "done" /\ accepted /\ ~SameIdentity /\ ~SeparateSlots
                         => AlternativeOnly
NoAmbiguousDecode == phase = "done" /\ accepted /\ ~SameIdentity /\ ~SeparateSlots
                     => DisjointFrames /\ SeparateSources
PreserveDisjointAlternatives == phase = "done" /\ AlternativeOnly /\ SeparateSources /\ DisjointFrames
                               => accepted
=============================================================================
