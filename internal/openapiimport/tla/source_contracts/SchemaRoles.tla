---------------------------- MODULE SchemaRoles ----------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES pending, visited, target, admitted, phase
vars == <<pending, visited, target, admitted, phase>>
Nodes == {"root", "component", "branch", "property", "boolean", "operation", "response", "inline", "dictionary", "required", "example", "fake"}
GrammarEdges == {<<"root", "component">>, <<"root", "operation">>, <<"operation", "response">>, <<"response", "inline">>, <<"component", "branch">>, <<"branch", "property">>, <<"component", "boolean">>}
SchemaRoles == {"component", "branch", "property", "boolean", "inline"}
Init == /\ pending = {"root"} /\ visited = {} /\ target \in Nodes
        /\ admitted = FALSE /\ phase = "walk"
Walk == /\ phase = "walk" /\ pending # {}
        /\ \E node \in pending:
          /\ visited' = visited \cup {node}
          /\ pending' = (pending \ {node}) \cup {next \in Nodes: <<node,next>> \in GrammarEdges /\ next \notin visited'}
        /\ UNCHANGED <<target,admitted,phase>>
Admit == /\ phase = "walk" /\ pending = {}
         /\ admitted' = CASE Policy = "component" -> target = "component"
                             [] Policy = "mapping" -> target \notin {"root","required"}
                             [] OTHER -> target \in visited /\ target \in SchemaRoles
         /\ phase' = "done"
         /\ UNCHANGED <<pending,visited,target>>
Next == Walk \/ Admit \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
ExactAdmission == phase = "done" => admitted = (target \in SchemaRoles)
=============================================================================
