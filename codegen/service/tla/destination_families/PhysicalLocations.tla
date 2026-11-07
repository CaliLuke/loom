-------------------------- MODULE PhysicalLocations ------------------------
EXTENDS Naturals
CONSTANT CarriesLocation
Contexts == {"source", "target"}
Destination(c) == IF c = "source" THEN "common" ELSE "wire"
Declaration(p) == CASE p = "common" -> "Pick2" [] p = "wire" -> "Pick3" [] OTHER -> "Pick"
VARIABLES phase, location, reference
vars == <<phase,location,reference>>
Init == /\ phase = [c \in Contexts |-> 0]
        /\ location = [c \in Contexts |-> "service"]
        /\ reference = [c \in Contexts |-> ""]
Enter(c) == /\ phase[c] = 0
            /\ phase' = [phase EXCEPT ![c] = 1]
            /\ location' = [location EXCEPT ![c] = IF CarriesLocation THEN Destination(c) ELSE @]
            /\ UNCHANGED reference
Child(c) == /\ phase[c] = 1
            /\ phase' = [phase EXCEPT ![c] = 2]
            /\ reference' = [reference EXCEPT ![c] = Declaration(location[c])]
            /\ UNCHANGED location
Next == (\E c \in Contexts : Enter(c) \/ Child(c)) \/ ((\A c \in Contexts : phase[c] = 2) /\ UNCHANGED vars)
ReferenceDeclared == \A c \in Contexts : phase[c] = 2 => reference[c] = Declaration(Destination(c))
=============================================================================
