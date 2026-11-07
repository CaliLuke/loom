------------------------- MODULE DeclarationEmission ------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS PackageGlobal, PackageLocal, IncludeServiceFiles
Contributions == {"a", "b", "c", "d", "e"}
Pkg(c) == CASE c \in {"a", "b"} -> "common" [] c = "c" -> "left" [] c = "d" -> "right" [] OTHER -> "other"
Symbol(c) == IF c \in {"c", "d"} THEN "Foo" ELSE "~union:Choice"
Svc(c) == CASE c = "a" -> "alpha" [] c = "b" -> "beta" [] c \in {"c", "d"} -> "gamma" [] OTHER -> "alpha"
FileKind(c) == IF c = "a" THEN "service" ELSE "custom"
File(c) == <<Pkg(c), FileKind(c), c>>
Physical(c) == <<Pkg(c), Symbol(c)>>
Declarations == {Physical(c) : c \in Contributions}
LocalKey(c) == IF PackageLocal THEN Physical(c) ELSE <<Symbol(c)>>
GlobalKey(c) == IF PackageGlobal THEN Physical(c) ELSE <<File(c), Symbol(c)>>
VARIABLES pending, localSeen, globalSeen, counts, placements
vars == <<pending, localSeen, globalSeen, counts, placements>>
Init == /\ pending = Contributions
        /\ localSeen = [s \in {"alpha", "beta", "gamma"} |-> {}]
        /\ globalSeen = {}
        /\ counts = [d \in Declarations |-> 0]
        /\ placements = [d \in Declarations |-> {}]
Emit(c) ==
 LET localOK == c \notin {"c", "d"} \/ LocalKey(c) \notin localSeen[Svc(c)]
     tracked == IncludeServiceFiles \/ FileKind(c) # "service"
     accepted == localOK /\ (~tracked \/ GlobalKey(c) \notin globalSeen)
 IN /\ c \in pending
    /\ pending' = pending \ {c}
    /\ localSeen' = [localSeen EXCEPT ![Svc(c)] = @ \cup {LocalKey(c)}]
    /\ globalSeen' = IF localOK /\ tracked THEN globalSeen \cup {GlobalKey(c)} ELSE globalSeen
    /\ counts' = IF accepted THEN [counts EXCEPT ![Physical(c)] = @ + 1] ELSE counts
    /\ placements' = IF accepted THEN [placements EXCEPT ![Physical(c)] = @ \cup {File(c)}] ELSE placements
Next == (\E c \in Contributions : Emit(c)) \/ (pending = {} /\ UNCHANGED vars)
Unique == \A d \in Declarations : counts[d] <= 1
Complete == pending = {} => (\A d \in Declarations : counts[d] = 1)
PlacementValid == \A d \in Declarations : placements[d] \subseteq {File(c) : c \in {x \in Contributions : Physical(x) = d}}
=============================================================================
