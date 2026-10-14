-------------------- MODULE DeclarationMerge --------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
VARIABLES scenario, pending, emitted, rejected
vars == <<scenario, pending, emitted, rejected>>
Item(n, identity, definition, effect) ==
  [n |-> n, category |-> "type", identity |-> identity,
   definition |-> definition, effect |-> effect]
Items(s) == CASE s = "distinct" -> {Item(1,"A","string",FALSE), Item(2,"B","int",FALSE)}
             [] s = "duplicate" -> {Item(1,"A","string",FALSE), Item(2,"A","string",FALSE)}
             [] s = "conflict" -> {Item(1,"A","string",FALSE), Item(2,"A","int",FALSE)}
             [] OTHER -> {Item(1,"","init",TRUE), Item(2,"","init",TRUE)}
Init == /\ scenario \in {"distinct", "duplicate", "conflict", "effects"}
        /\ pending = Items(scenario)
        /\ emitted = {}
        /\ rejected = FALSE
Step == /\ pending # {}
        /\ \E next \in pending:
          LET matches == {old \in emitted:
                IF Mode = "Category" THEN old.category = next.category
                ELSE ~next.effect /\ old.identity = next.identity}
              conflict == Mode = "Identity" /\
                \E old \in matches: old.definition # next.definition
          IN /\ pending' = pending \ {next}
             /\ rejected' = (rejected \/ conflict)
             /\ emitted' = IF matches = {} THEN emitted \cup {next} ELSE emitted
             /\ UNCHANGED scenario
Spec == Init /\ [][Step]_vars
ConflictAdmission == pending = {} => (rejected = (scenario = "conflict"))
DeclarationCoverage == pending = {} /\ ~rejected /\ scenario # "effects" =>
  {item.identity: item \in emitted} = {item.identity: item \in Items(scenario)}
OccurrencePreservation == pending = {} /\ scenario = "effects" => Cardinality(emitted) = 2
NoDuplicateDeclaration == scenario # "effects" =>
  Cardinality(emitted) = Cardinality({item.identity: item \in emitted})
=================================================================
