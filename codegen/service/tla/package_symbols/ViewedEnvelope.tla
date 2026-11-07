------------------------- MODULE ViewedEnvelope -----------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT OwnedEnvelope
VARIABLES scenario, occupied, mapName, envelope, declaration, reference, phase
vars == <<scenario, occupied, mapName, envelope, declaration, reference, phase>>
Sources == {"Foo", "FooView", "viewed(Foo)"}
ProjectedKey(s) == s \o "View"
EnvelopeKey(s) == "viewed(" \o s \o ")"
PhysicalKeysDisjoint == \A a,b \in Sources : EnvelopeKey(a) # ProjectedKey(b)
Candidates(n) == <<n, n \o "2", n \o "3", n \o "4">>
Family(n) == {n, "Validate" \o n}
Fresh(n, used, family) ==
    LET choices == Candidates(n)
        free == {i \in 1..Len(choices) :
                 (IF family THEN Family(choices[i]) ELSE {choices[i]}) \cap used = {}}
        first == CHOOSE i \in free : \A j \in free : i <= j
    IN choices[first]
Init == /\ scenario \in {"cross-result", "real-type", "map"}
        /\ occupied = CASE scenario = "real-type" -> {"FooBar"}
                         [] scenario = "map" -> {"FooMap"}
                         [] OTHER -> {}
        /\ mapName = "" /\ envelope = "" /\ declaration = "" /\ reference = ""
        /\ phase = "map"
Metadata ==
    /\ phase = "map"
    /\ mapName' = IF OwnedEnvelope THEN Fresh("FooMap", occupied, FALSE) ELSE "FooMap"
    /\ occupied' = occupied \cup
         (IF OwnedEnvelope THEN {mapName'} ELSE {"Foo", "ValidateFoo", "ValidateFooBar"})
    /\ phase' = "envelope"
    /\ UNCHANGED <<scenario, envelope, declaration, reference>>
Envelope ==
    /\ phase = "envelope"
    /\ envelope' = Fresh("FooBar", occupied, TRUE)
    /\ declaration' = IF OwnedEnvelope THEN envelope' ELSE "FooBar"
    /\ occupied' = occupied \cup Family(envelope')
    /\ phase' = "reference"
    /\ UNCHANGED <<scenario, mapName, reference>>
Reference == /\ phase = "reference" /\ reference' = envelope /\ phase' = "done"
             /\ UNCHANGED <<scenario, occupied, mapName, envelope, declaration>>
Next == Metadata \/ Envelope \/ Reference \/ (phase = "done" /\ UNCHANGED vars)
ReferenceDeclared == reference # "" => reference = declaration
MapOwned == phase # "map" /\ scenario = "map" => mapName # "FooMap"
=============================================================================
