--------------------------- MODULE Credentials ---------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES a, b, conjunctive, checks, grouped, roles, targetAnd
vars == <<a,b,conjunctive,checks,grouped,roles,targetAnd>>

\* Names are canonical slot names; actual HTTP case folding is tested in Go.
Inputs == {
 [role |-> "JWT", location |-> "header", name |-> "authorization", codec |-> "bearer", part |-> "value"],
 [role |-> "OAuth", location |-> "header", name |-> "authorization", codec |-> "bearer", part |-> "value"],
 [role |-> "BasicUser", location |-> "header", name |-> "authorization", codec |-> "basic", part |-> "username"],
 [role |-> "BasicPass", location |-> "header", name |-> "authorization", codec |-> "basic", part |-> "password"],
 [role |-> "KeyHeader", location |-> "header", name |-> "key", codec |-> "raw", part |-> "value"],
 [role |-> "KeyHeader2", location |-> "header", name |-> "key", codec |-> "raw", part |-> "value"],
 [role |-> "KeyQuery", location |-> "query", name |-> "key", codec |-> "raw", part |-> "value"]}
Physical(x) == <<x.location,x.name,x.codec,x.part>>
Key(x) == CASE Mode = "scheme" -> x.role
              [] Mode = "slot" -> <<x.location,x.name>>
              [] Mode = "codec" -> <<x.codec,x.part>>
              [] OTHER -> Physical(x)
Init == /\ a \in Inputs /\ b \in Inputs
        /\ conjunctive \in BOOLEAN
        /\ checks \in SUBSET {a.role,b.role}
        /\ grouped = FALSE /\ roles = {} /\ targetAnd = FALSE
Group == /\ ~grouped
         /\ grouped' = TRUE
         /\ roles' = IF Mode = "drop-role" /\ Key(a)=Key(b) THEN {a.role} ELSE {a.role,b.role}
         /\ targetAnd' = IF Mode = "flatten" THEN FALSE ELSE conjunctive
         /\ UNCHANGED <<a,b,conjunctive,checks>>
Next == Group \/ (grouped /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
SameRepresentationOneSource == grouped => ((Key(a)=Key(b)) <=> (Physical(a)=Physical(b)))
AllRolesRetained == grouped => roles = {a.role,b.role}
Allowed(andMode, required) == IF andMode THEN required \subseteq checks ELSE required \cap checks # {}
RequirementIntent == grouped => Allowed(conjunctive,{a.role,b.role}) = Allowed(targetAnd,roles)
=============================================================================
