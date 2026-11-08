-------------------- MODULE DiscriminatorReachability --------------------
EXTENDS FiniteSets, Sequences, TLC

CONSTANTS Mode, Cases
VARIABLES scenario, renamed, phase, schemas, refs, mappings, defaults,
          dependencies, reachable, pending

vars == <<scenario, renamed, phase, schemas, refs, mappings, defaults,
          dependencies, reachable, pending>>

Nodes == {"P", "C", "G", "U", "V", "M", "B", "A1", "A2"}
AliasNodes == {"A1", "A2"}
Canon(n) == IF n \in AliasNodes THEN "M" ELSE n
Name(n) == IF renamed THEN n \o "_assigned" ELSE n
Names(ns) == {Name(n) : n \in ns}
EdgesNamed(es) == {<<Name(e[1]), Name(e[2])>> : e \in es}
EdgesCanonical(es) == {<<Canon(e[1]), Canon(e[2])>> : e \in es}
NamedAliases == Names(AliasNodes)
Resolve(n) == IF n \in NamedAliases THEN Name("M") ELSE n
ResolveEdges(es) == {<<e[1], Resolve(e[2])>> : e \in es}

(* Directed pair <child,parent> is authored direct composition identity.
   Only the explicit reference marker produces an allOf reference. *)
Compositions ==
  IF scenario \in {"implicit", "wrapped", "cycle", "combined", "unused"}
  THEN {<<"C", "P">>, <<"G", "C">>, <<"V", "U">>}
       \cup (IF scenario = "combined" THEN {<<"M", "P">>} ELSE {})
       \cup (IF scenario = "wrapped" THEN {<<"P", "B">>} ELSE {})
  ELSE IF scenario = "flat" THEN {<<"C", "P">>, <<"V", "U">>}
  ELSE {<<"M", "P">>, <<"V", "U">>}

AllOfEdges ==
  IF scenario \in {"flat", "mapped", "default"}
  THEN {<<"V", "U">>}
  ELSE Compositions \ {<<"M", "P">>}

ParentHints == {"P", "U"}
WrappedHints == IF scenario = "wrapped" THEN {"P"} ELSE {}
StructuralEdges == AllOfEdges \cup
  IF scenario = "cycle" THEN {<<"P", "C">>} ELSE {}
MappingEdges == IF scenario \in {"mapped", "combined"}
                THEN {<<"P", "A1">>} ELSE {}
DefaultEdges == IF scenario \in {"default", "combined"}
                THEN {<<"P", "A1">>} ELSE {}
AliasEdges == {<<"A1", "A2">>, <<"A2", "M">>}
RootNodes == IF scenario = "unused" THEN {"B"} ELSE {"P"}

Reverse(es, parents) ==
  {<<e[2], e[1]>> : e \in {edge \in es : edge[2] \in parents}}
SemanticDependencies == Reverse(AllOfEdges, ParentHints)
DraftParents == IF Mode = "draft" THEN ParentHints \ WrappedHints ELSE ParentHints
CapturedDependencies ==
  IF Mode = "old" THEN {}
  ELSE Reverse(IF Mode = "allBases" THEN Compositions ELSE AllOfEdges,
               DraftParents)

RECURSIVE Closure(_, _)
Closure(seeds, edges) ==
  LET more == seeds \cup {e[2] : e \in {edge \in edges : edge[1] \in seeds}}
  IN IF more = seeds THEN seeds ELSE Closure(more, edges)

ExpectedEdges == EdgesCanonical(StructuralEdges \cup MappingEdges \cup
                                DefaultEdges \cup SemanticDependencies)
Expected == Names(Closure({Canon(n) : n \in RootNodes}, ExpectedEdges))
Retained == reachable \cap schemas

Init ==
  /\ scenario \in Cases
  /\ renamed \in BOOLEAN
  /\ phase = "allocate"
  /\ schemas = {}
  /\ refs = {}
  /\ mappings = {}
  /\ defaults = {}
  /\ dependencies = {}
  /\ reachable = {}
  /\ pending = {}

Allocate ==
  /\ phase = "allocate"
  /\ schemas' = Names(Nodes)
  /\ refs' = EdgesNamed(StructuralEdges \cup AliasEdges)
  /\ mappings' = EdgesNamed(MappingEdges)
  /\ defaults' = EdgesNamed(DefaultEdges)
  /\ dependencies' = IF Mode = "staleNames" THEN CapturedDependencies
                      ELSE EdgesNamed(CapturedDependencies)
  /\ phase' = "collapse"
  /\ UNCHANGED <<scenario, renamed, reachable, pending>>

CollapseAliases ==
  /\ phase = "collapse"
  /\ schemas' = schemas \ NamedAliases
  /\ refs' = ResolveEdges({e \in refs : e[1] \notin NamedAliases})
  /\ mappings' = IF Mode \in {"old", "noMappingRewrite"}
                  THEN mappings ELSE ResolveEdges(mappings)
  /\ defaults' = ResolveEdges(defaults)
  /\ reachable' = Names(RootNodes)
  /\ pending' = Names(RootNodes)
  /\ phase' = "walk"
  /\ UNCHANGED <<scenario, renamed, dependencies>>

Visit ==
  /\ phase = "walk"
  /\ \E node \in pending:
       LET ordinary == IF node \in schemas
                       THEN refs \cup defaults \cup
                            IF Mode \in {"old", "noMappingTraversal"}
                            THEN {} ELSE mappings
                       ELSE {}
           edges == ordinary \cup dependencies
           targets == {e[2] : e \in {edge \in edges : edge[1] = node}}
       IN /\ reachable' = reachable \cup targets
          /\ pending' = (pending \ {node}) \cup (targets \ reachable)
  /\ UNCHANGED <<scenario, renamed, phase, schemas, refs, mappings, defaults,
                 dependencies>>

Finish ==
  /\ phase = "walk"
  /\ pending = {}
  /\ phase' = "done"
  /\ UNCHANGED <<scenario, renamed, schemas, refs, mappings, defaults,
                 dependencies, reachable, pending>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Allocate \/ CollapseAliases \/ Visit \/ Finish \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

TypeOK ==
  /\ scenario \in Cases
  /\ renamed \in BOOLEAN
  /\ phase \in {"allocate", "collapse", "walk", "done"}
  /\ schemas \subseteq Names(Nodes)
  /\ reachable \subseteq Names(Nodes) \cup Nodes
  /\ pending \subseteq reachable

CompletePublication == phase = "done" => Expected \subseteq Retained
NoExtraneousPublication == phase = "done" => Retained \subseteq Expected
NoDanglingHintTargets ==
  phase = "done" =>
    \A edge \in mappings \cup defaults:
      edge[1] \in Retained => edge[2] \in Retained
DirectDependenciesOnly ==
  phase = "done" =>
    dependencies \subseteq EdgesNamed(SemanticDependencies)
Terminates == <> (phase = "done")
=============================================================================
