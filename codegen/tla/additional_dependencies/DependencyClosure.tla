------------------------- MODULE DependencyClosure -------------------------
EXTENDS FiniteSets
CONSTANT IncludeAdditional
VARIABLES pending, seen
vars == <<pending, seen>>
Nodes == {"record", "declared", "carrier", "union", "left", "right", "scalar"}
Ordinary == {<<"record", "declared">>, <<"carrier", "union">>,
             <<"union", "left">>, <<"union", "right">>,
             <<"left", "scalar">>, <<"right", "record">>}
Additional == {<<"record", "carrier">>}
Edges == Ordinary \cup (IF IncludeAdditional THEN Additional ELSE {})
Children(node) == {child \in Nodes : <<node, child>> \in Edges}
Init == /\ pending = {"record"}
        /\ seen = {}
Next == \E node \in pending:
          /\ seen' = seen \cup {node}
          /\ pending' = (pending \cup Children(node)) \ seen'
Spec == Init /\ [][Next]_vars
TypeOK == /\ pending \subseteq Nodes
          /\ seen \subseteq Nodes
          /\ pending \cap seen = {}
Complete == pending = {} => seen = Nodes
=============================================================================
