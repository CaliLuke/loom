-------------------- MODULE TransformDiscovery --------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES phase, order, aliasCollision, canonical, effects, failures,
          concrete, names, cursor, emitted
vars == <<phase, order, aliasCollision, canonical, effects, failures,
          concrete, names, cursor, emitted>>

Nodes == {"Root", "A", "B", "Plain"}
Keys == Nodes \cup {"mappedA", "mappedB", "concreteA", "concreteB",
                   "freshA", "freshB"}
None == "none"
Orders == {<<"Root", "A", "B", "Plain">>,
           <<"Root", "B", "A", "Plain">>}
Children(n) == CASE n = "Root" -> {"A", "B", "Plain"}
                 [] n = "A" -> {"A"}
                 [] OTHER -> {}
LocalEffect(n) == n \in {"A", "B"}
LocalFailure(n) == n = "B"
Default(n) == CASE n = "A" -> "body9"
                [] n = "B" -> "body11"
                [] OTHER -> None
Physical(n) == IF n \in {"A", "B"} THEN "Node" ELSE n
ExpectedEffect(n) == n # "Plain"
ExpectedFailure(n) == n \in {"Root", "B"}
Rank(n, o) == CHOOSE i \in 1..Len(o) : o[i] = n
ExpectedName(n, o) ==
 IF n = "Plain" THEN "ordinaryPlain"
 ELSE IF n = "Root" THEN "admittedRoot"
 ELSE IF Rank(n, o) < Rank(IF n = "A" THEN "B" ELSE "A", o)
      THEN "admittedNode" ELSE "admittedNode2"

Requests == <<[origin |-> "Root", role |-> "original"],
              [origin |-> "A", role |-> "mapped"],
              [origin |-> "B", role |-> "mapped"],
              [origin |-> "A", role |-> "concrete"],
              [origin |-> "B", role |-> "concrete"],
              [origin |-> "A", role |-> "recursive"],
              [origin |-> "B", role |-> "original"],
              [origin |-> "Plain", role |-> "original"]>>

ConcreteKey(n) ==
 IF n = "A" THEN IF aliasCollision THEN "B" ELSE "concreteA"
 ELSE IF n = "B" THEN "concreteB" ELSE n
MappedKey(n) == IF n = "A" THEN "mappedA" ELSE "mappedB"
FreshKey(n) == IF n = "A" THEN "freshA" ELSE "freshB"

Lookup(r) ==
 IF r.role = "mapped" /\ Mode = "mapped-copy"
 THEN canonical[MappedKey(r.origin)]
 ELSE IF r.role = "concrete" THEN
   IF Mode = "nullable-copy" THEN canonical[FreshKey(r.origin)]
   ELSE IF Mode = "alias-overwrite" THEN canonical[concrete[r.origin].key]
   ELSE concrete[r.origin].owner
 ELSE canonical[r.origin]

Init ==
 /\ phase = "discover"
 /\ order \in Orders
 /\ aliasCollision \in BOOLEAN
 /\ canonical = [k \in Keys |-> IF k \in Nodes THEN k ELSE None]
 /\ effects = [n \in Nodes |-> LocalEffect(n)]
 /\ failures = [n \in Nodes |-> LocalFailure(n)]
 /\ concrete = [n \in Nodes |-> [owner |-> None, key |-> None]]
 /\ names = [n \in Nodes |-> None]
 /\ cursor = 1
 /\ emitted = <<>>

CloseGraph ==
 /\ phase = "discover"
 /\ LET nextEffects == [n \in Nodes |-> effects[n] \/
                              (\E c \in Children(n) : effects[c])]
        nextFailures == [n \in Nodes |-> failures[n] \/
                               (\E c \in Children(n) : failures[c])]
    IN /\ effects' = nextEffects
       /\ failures' = nextFailures
       /\ phase' = IF nextEffects = effects /\ nextFailures = failures
                    THEN "copies" ELSE "discover"
 /\ UNCHANGED <<order, aliasCollision, canonical, concrete, names, cursor, emitted>>

BindCopies ==
 /\ phase = "copies"
 /\ concrete' = [n \in Nodes |-> [owner |-> n, key |-> ConcreteKey(n)]]
 /\ canonical' =
      IF Mode = "alias-overwrite" THEN
        [k \in Keys |-> IF k = ConcreteKey("A") THEN "A"
                       ELSE IF k = ConcreteKey("B") THEN "B"
                       ELSE canonical[k]]
      ELSE canonical
 /\ phase' = "allocate"
 /\ UNCHANGED <<order, aliasCollision, effects, failures, names, cursor, emitted>>

Allocate ==
 /\ phase = "allocate"
 /\ names' = [n \in Nodes |-> IF Mode = "physical-only"
                              THEN Physical(n) ELSE ExpectedName(n, order)]
 /\ phase' = "emit"
 /\ UNCHANGED <<order, aliasCollision, canonical, effects, failures,
                concrete, cursor, emitted>>

Emit ==
 /\ phase = "emit"
 /\ LET r == Requests[cursor]
        owner == Lookup(r)
        entry == [origin |-> r.origin, role |-> r.role, owner |-> owner,
                  selected |-> IF owner \in Nodes THEN Default(owner) ELSE None,
                  fallible |-> IF owner \in Nodes THEN failures[owner] ELSE FALSE,
                  helper |-> IF owner \in Nodes THEN names[owner] ELSE None]
    IN emitted' = Append(emitted, entry)
 /\ cursor' = cursor + 1
 /\ phase' = IF cursor = Len(Requests) THEN "done" ELSE "emit"
 /\ UNCHANGED <<order, aliasCollision, canonical, effects, failures, concrete, names>>

Next == CloseGraph \/ BindCopies \/ Allocate \/ Emit
Spec == Init /\ [][Next]_vars

CanonicalPreserved == \A n \in Nodes : canonical[n] = n
LookupComplete == \A i \in 1..Len(emitted) : emitted[i].owner \in Nodes
SelectedOwner == \A i \in 1..Len(emitted) :
  emitted[i].owner = emitted[i].origin /\
  emitted[i].selected = Default(emitted[i].origin)
FailureCorrespondence == \A i \in 1..Len(emitted) :
  emitted[i].fallible = ExpectedFailure(emitted[i].origin)
FixedOrderNames == \A i \in 1..Len(emitted) :
  emitted[i].helper = ExpectedName(emitted[i].origin, order)
DistinctSemanticHelpers == phase \in {"emit", "done"} => names["A"] # names["B"]
ClosureComplete == phase \in {"copies", "allocate", "emit", "done"} =>
  \A n \in Nodes : effects[n] = ExpectedEffect(n) /\ failures[n] = ExpectedFailure(n)
RecursiveIdentity == \A i, j \in 1..Len(emitted) :
  emitted[i].origin = emitted[j].origin => emitted[i].helper = emitted[j].helper
ControlledCopy == phase \in {"allocate", "emit", "done"} =>
  \A n \in Nodes : concrete[n].owner = n /\ concrete[n].key = ConcreteKey(n)
===================================================================
