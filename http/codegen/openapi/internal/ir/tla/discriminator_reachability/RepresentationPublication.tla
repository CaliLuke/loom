------------------- MODULE RepresentationPublication -------------------
(* Representation extension of DiscriminatorReachability. A declaration has a
   raw registration representation and a JSON representation. The JSON
   representation is a separate component when the JSON codec changes the
   declaration's own byte schemas or those of its discriminator publication
   closure (implicit children, mapping and default targets). The allocation
   quotient later merges component bodies; occurrence codec context is kept
   apart from that quotient and decided before it.

   An occurrence reaches a declaration through one of several shapes: the
   declaration itself, an HTTP body copy or a flattened union body whose plan
   still targets the declaration, a nested field whose type HTTP lowering
   renamed into a copy, or an inline composition contributor of a field, of a
   field nested in a declared or inline holder, of an inline composition's own
   body, or of the root. The occurrence must see the declaration's context in
   every shape. Projection decides at the occurrence root from the occurrence
   attribute graph and at every deeper member from the value-plan node, whose
   owned attribute copy carries no composition contributors. Context counts
   only at nodes whose codec is JSON; a form occurrence is a separate path.

   A form or multipart body is a non-JSON root whose object fields are
   JSON-encoded on the wire. Its baseline is acquired from value-plan copies,
   so the composition contributors of its descendants are lost unless they are
   read from the paired original graph (GraphAcquisition). The plan codec of a
   JSON-encoded descendant is the form codec unless the plan records the JSON
   boundary (JSONBoundary); the wire codec is JSON either way. *)
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANTS Mode, Cases
VARIABLES scenario, phase, retainedIR, reachable, pending

vars == <<scenario, phase, retainedIR, reachable, pending>>

Decls == {"P", "C", "T", "D", "U"}

(* Occurrence-shape scenarios reuse a base scenario's declarations. "decl" is
   the occurrence shape of every other scenario. *)
FormScenarios == {"formJSONHolder", "formJSONField", "formJSONNamed", "formJSONItems", "formMixed"}
OccurrenceScenarios == {"bodyCopy", "flatUnion", "nestedField", "inlineField", "inlineRoot", "inlineDeep",
                       "inlineHolder", "inlineNested", "unionHolder", "mixedInline"} \cup FormScenarios
kind == CASE scenario = "flatUnion" -> "unionFallback"
          [] scenario \in {"mixedInline", "formMixed"} -> "plainMixed"
          [] scenario \in OccurrenceScenarios -> "plainByteChild"
          [] OTHER -> scenario
Occurrence == IF scenario \in OccurrenceScenarios THEN scenario ELSE "decl"

(* P is a hinted parent with direct allOf child C, explicit mapping target T
   and defaultMapping target D. U is an untagged union hint whose separate
   fallback is D. "refined": C narrows an inherited byte constraint. "form",
   "used", "plainMixed": P also has a non-JSON occurrence; "used": C has its
   own JSON occurrence. "plain*" scenarios give P no byte content. *)
BytePlain == {"plainByteChild", "plainMixed", "plainMapped", "plainDefault", "unionFallback", "plain"}
OwnBytes(x) ==
  CASE x = "P" -> kind \notin BytePlain
    [] x = "C" -> kind \notin {"plain", "plainMapped", "plainDefault", "unionFallback"}
    [] x = "T" -> kind \notin {"plain", "plainByteChild", "plainMixed", "plainDefault", "unionFallback"}
    [] x = "D" -> kind \notin {"plain", "plainByteChild", "plainMixed", "plainMapped"}
    [] OTHER   -> FALSE
Mapped == kind \in {"mapped", "plainMapped"}
Defaulted == kind \in {"default", "plainDefault"}
Deps(x) ==
  CASE x = "P" -> {"C"} \cup (IF Mapped THEN {"T"} ELSE {}) \cup (IF Defaulted THEN {"D"} ELSE {})
    [] x = "U" -> IF kind = "unionFallback" THEN {"D"} ELSE {}
    [] OTHER   -> {}
Bases(x) == IF x = "C" THEN {"P"} ELSE {}

RECURSIVE Closure(_, _)
Closure(seeds, edges) ==
  LET more == seeds \cup {e[2] : e \in {edge \in edges : edge[1] \in seeds}}
  IN IF more = seeds THEN seeds ELSE Closure(more, edges)

(* Representation context closure: discriminator publication dependencies and
   composition contributors, transitively. A child's JSON representation is
   distinct when its contributor's is, because its allOf names that parent. *)
Neighbors(x, contributors) == Deps(x) \cup (IF contributors THEN Bases(x) ELSE {})
ContextEdges(contributors) == {<<x, y>> \in Decls \X Decls : y \in Neighbors(x, contributors)}
ClosureBytes(x, contributors) ==
  \E y \in Closure(Neighbors(x, contributors), ContextEdges(contributors)): OwnBytes(y)

(* Whether the JSON representation of x is a distinct component, by design:
   "own" (R3) uses x's body only; "publication" adds discriminator publication
   targets; "full" (R4 onward) adds composition contributors too. *)
Design == CASE Mode \in {"r2", "r3"} -> "own"
            [] Mode = "noContributorContext" -> "publication"
            [] OTHER -> "full"
JSONDistinct(x, design) == OwnBytes(x) \/ (design # "own" /\ ClosureBytes(x, design = "full"))
NodeOf(x, r, design) ==
  IF r = "raw" \/ ~JSONDistinct(x, design) THEN <<x, "raw">> ELSE <<x, "json">>
Node(x, r) == NodeOf(x, r, Design)
Rep(n) == IF n[2] = "raw" THEN "raw" ELSE "json"
ChildPlanParent == <<"P", "childplan">>

(* The value-plan nodes an occurrence visits from its root to the node that
   refers to the context-bearing declaration, as lowered by HTTP. Each node
   has the physical named type of its attribute ("decl": a declaration root;
   "copy": an HTTP body copy or a renamed nested copy whose ID is not a
   declaration root; "none": no named type), the plan's target declaration
   (TargetDeclarationID; "none" for an inline value), the occurrence
   attribute's own composition contributors, the node's plan codec and its
   wire codec. "H" is the holder declaration of the field scenarios, "W" the
   wrapper of the deep and inline-holder scenarios, "B" a plain declared
   contributor of an inline composition and "X" a union declaration. *)
N(named, target, contributors) ==
  [named |-> named, target |-> target, contributors |-> contributors, codec |-> "json", wire |-> "json"]
(* Whether the plan records the JSON boundary of a JSON-encoded descendant of a
   form root. Without it the descendant inherits the form plan codec. *)
JSONBoundary == Mode \notin {"v4", "graphAcquisition"}
(* A node below a form root with wire codec wire: the form root itself
   ("form") or a JSON-encoded field or one of its descendants ("json"). *)
NF(named, target, contributors, wire) ==
  [named |-> named, target |-> target, contributors |-> contributors,
   codec |-> IF wire = "json" /\ JSONBoundary THEN "json" ELSE "form", wire |-> wire]
OccurrencePath ==
  CASE Occurrence = "bodyCopy" -> <<N("copy", "P", {})>>
    [] Occurrence = "flatUnion" -> <<N("none", "U", {})>>
    [] Occurrence = "nestedField" -> <<N("copy", "H", {}), N("copy", "P", {})>>
    [] Occurrence = "inlineField" -> <<N("copy", "H", {}), N("none", "none", {"P"})>>
    [] Occurrence = "inlineDeep" -> <<N("copy", "W", {}), N("copy", "H", {}), N("none", "none", {"P"})>>
    [] Occurrence = "inlineRoot" -> <<N("copy", "none", {"P"})>>
    [] Occurrence \in {"inlineHolder", "mixedInline"} ->
         <<N("copy", "W", {}), N("none", "none", {}), N("none", "none", {"P"})>>
    [] Occurrence = "inlineNested" -> <<N("copy", "W", {}), N("none", "none", {"B"}), N("none", "none", {"P"})>>
    [] Occurrence = "unionHolder" -> <<N("none", "X", {}), N("none", "none", {}), N("none", "none", {"P"})>>
    [] Occurrence = "formJSONHolder" ->
         <<NF("copy", "none", {}, "form"), NF("none", "none", {}, "json"), NF("none", "none", {"P"}, "json")>>
    [] Occurrence \in {"formJSONField", "formMixed"} -> <<NF("copy", "none", {}, "form"), NF("none", "none", {"P"}, "json")>>
    [] Occurrence = "formJSONNamed" ->
         <<NF("copy", "none", {}, "form"), NF("copy", "H", {}, "json"), NF("none", "none", {"P"}, "json")>>
    (* Item-applied JSON content: the array container keeps the form codec and
       each item is a JSON content boundary. *)
    [] Occurrence = "formJSONItems" ->
         <<NF("copy", "none", {}, "form"), NF("none", "none", {}, "form"), NF("none", "none", {"P"}, "json")>>
    [] OTHER -> <<N("decl", IF kind = "unionFallback" THEN "U" ELSE "P", {})>>
(* A non-JSON root's baseline is acquired from value-plan copies. Without
   GraphAcquisition the composition contributors of every node below it are
   lost, so the authored composition reference is never emitted. *)
ParameterAcquisition == OccurrencePath[1].wire = "form"
GraphAcquisition == Mode \notin {"v4", "jsonBoundaryOnly"}
AcquiredPath ==
  [i \in 1..Len(OccurrencePath) |->
     IF ParameterAcquisition /\ ~GraphAcquisition
     THEN [OccurrencePath[i] EXCEPT !.contributors = {}] ELSE OccurrencePath[i]]
(* The form occurrence of the mixed scenarios: a form body that is P itself,
   whose fields are form fields. (A P-typed field of a form body is JSON
   content, an occurrence like formJSONField.) Boundaries attach to the
   normalized occurrence that each field, branch or map value owns privately;
   a boundary attached to a declaration shared with this occurrence
   (SharedBoundary, the countercontrol) leaks the JSON codec into it. *)
SharedBoundary == Mode = "sharedBoundary"
FormPath == <<[named |-> "copy", target |-> "P", contributors |-> {},
               codec |-> IF SharedBoundary THEN "json" ELSE "form"]>>
OccurrenceNodes == {AcquiredPath[i] : i \in 1..Len(AcquiredPath)}
(* Declarations whose publication neighbors lie inside a declaration's graph:
   the holders contain P through a typed field or an inline contributor. *)
DeclarationGraph(t) == CASE t = "H" -> {"P"} [] t \in {"W", "X"} -> {"H", "P"} [] OTHER -> {}

(* Root resolution from the occurrence attribute graph. R4 resolved only a
   physical named type that is itself a declaration root, and read
   contributors only from the attributes of named types. *)
TargetOnlyFromDeclarations == Mode \in {"r4", "noTargetDeclaration"}
ContributorsOnlyFromNamed == Mode \in {"r4", "noInlineContributors"}
(* Context is added only at JSON codec nodes; a codec-blind design (the
   countercontrol) also adds it at other nodes. *)
CodecBlind == Mode = "codecBlind"
CodecAllows(node) == CodecBlind \/ node.codec = "json"
Resolved(node) ==
  IF ~CodecAllows(node) THEN {}
  ELSE (IF ~TargetOnlyFromDeclarations \/ node.named = "decl" THEN {node.target} ELSE {})
       \cup (IF ~ContributorsOnlyFromNamed \/ node.named # "none" THEN node.contributors ELSE {})
RootSees == \E node \in OccurrenceNodes: Resolved(node) \cap {"P", "U"} # {}

(* Member decisions from value-plan nodes. Plan targets survive in the plan
   nodes of the whole subtree; the owned attribute copies carry no composition
   contributors. P1 (construction context) lets a member consult the
   contributors that constructions record: "position" reads only the member's
   own value position (v3); "descendant" pairs the member's construction with
   its plan subtree, as projection does, and reads the contributors recorded at
   every descendant JSON position. P2 (DeclarationGraphContext) lets a node
   that denotes a declaration see the publication neighbors of that
   declaration's whole graph. R5 had neither. *)
SlotScope == CASE Mode \in {"r4", "r5", "declarationGraphOnly"} -> "none"
               [] Mode \in {"constructionOnly", "v3"} -> "position"
               [] OTHER -> "descendant"
DeclarationGraphContext == Mode \notin {"r4", "r5", "constructionOnly"}
PlanTarget(node) ==
  IF ~CodecAllows(node) THEN {}
  ELSE IF ~TargetOnlyFromDeclarations \/ node.named = "decl" THEN {node.target} ELSE {}
RecordedContributors(j) ==
  IF CodecAllows(AcquiredPath[j]) THEN AcquiredPath[j].contributors ELSE {}
SlotResolved(i) ==
  UNION {PlanTarget(AcquiredPath[j]) \cup
           (IF DeclarationGraphContext THEN UNION {DeclarationGraph(t) : t \in PlanTarget(AcquiredPath[j])} ELSE {})
         : j \in i..Len(AcquiredPath)}
  \cup (CASE SlotScope = "position" -> RecordedContributors(i)
           [] SlotScope = "descendant" -> UNION {RecordedContributors(j) : j \in i..Len(AcquiredPath)}
           [] OTHER -> {})
SlotSees(i) == SlotResolved(i) \cap {"P", "U"} # {}
(* A parameter-acquired root analyzes the plan copy, so its decision reads
   plan targets, declaration graphs and recorded constructions like a member. *)
DecisionRootSees == IF ParameterAcquisition THEN SlotSees(1) ELSE RootSees
(* Whether the form occurrence acquires JSON context through the same
   codec-gated resolution: only a codec-blind design resolves anything at its
   non-JSON nodes. *)
FormSeen == \E i \in 1..Len(FormPath): Resolved(FormPath[i]) \cap {"P"} # {}

(* Whether the JSON occurrence sees the context of the declaration whose
   publication closure matters: the root decides from the attribute graph, and
   no deeper member on the path returns its raw baseline. *)
ContextSeen == DecisionRootSees /\ \A i \in 2..Len(OccurrencePath): SlotSees(i)
(* A JSON occurrence that misses the context still projects the declaration's
   own byte fields, which its physical graph carries. *)
JSONNode(x, design, seen) == IF seen THEN NodeOf(x, "json", design) ELSE NodeOf(x, "json", "own")

(* How the occurrence references its context-bearing declaration: "none" when
   a form occurrence lost the composition reference that is its only edge,
   "raw" when the referencing node's plan codec is not JSON (the composition
   contributor stays the raw registration), and "json" otherwise. *)
ReferencesFamily ==
  Occurrence \notin FormScenarios \/ \E j \in 1..Len(AcquiredPath): "P" \in AcquiredPath[j].contributors
OccurrenceRef ==
  IF ~ReferencesFamily THEN "none"
  ELSE IF AcquiredPath[Len(AcquiredPath)].codec = "json" THEN "json" ELSE "raw"
OccurrenceRoot(design, seen, ref) ==
  CASE ref = "none" -> {}
    [] kind = "unionFallback" -> {JSONNode("U", design, seen)}
    [] ref = "raw" -> {NodeOf("P", "raw", design)}
    [] OTHER -> {JSONNode("P", design, seen)}

(* A non-JSON occurrence publishes the raw representation unless a design
   wrongly gives it JSON context. *)
RootsOf(design, seen, formJSON, ref) ==
  OccurrenceRoot(design, seen, ref)
  \cup (IF kind \in {"form", "used", "plainMixed"}
        THEN {IF formJSON THEN NodeOf("P", "json", design) ELSE NodeOf("P", "raw", design)} ELSE {})
  \cup (IF kind = "used" THEN {NodeOf("C", "json", design)} ELSE {})
Roots == RootsOf(Design, ContextSeen, FormSeen, OccurrenceRef)
(* Every occurrence path ends at a node whose wire codec is JSON, so whatever
   it references is published through a JSON occurrence. *)
JSONRoots ==
  OccurrenceRoot(Design, ContextSeen, OccurrenceRef)
  \cup (IF kind = "used" THEN {Node("C", "json")} ELSE {})

ChildPlanBase == Mode \in {"r2", "childPlanBase"}
RawHintRefs == Mode \in {"r2", "rawHintRefs"}
RawDependenciesOnly == Mode = "r2"
IRFollowsDependencies == Mode \notin {"r2", "noIRDeps"}
BodyQuotient == Mode = "bodyQuotient"

BaseOf(r, childPlan, design) ==
  IF r = "json" /\ JSONDistinct("P", design) /\ childPlan /\ kind = "refined"
  THEN ChildPlanParent ELSE NodeOf("P", r, design)
AllOfEdgesOf(childPlan, design) ==
  {<<NodeOf("C", r, design), BaseOf(r, childPlan, design)>> : r \in {"raw", "json"}}
ParentNodesOf(design) == {NodeOf("P", "raw", design), NodeOf("P", "json", design), ChildPlanParent}
UnionNodesOf(design) == {NodeOf("U", "raw", design), NodeOf("U", "json", design)}
HintEdgesOf(raw, design) ==
  LET target(x, p) == IF raw THEN NodeOf(x, "raw", design) ELSE NodeOf(x, Rep(p), design)
  IN (IF Mapped THEN {<<p, target("T", p)>> : p \in ParentNodesOf(design)} ELSE {})
     \cup (IF Defaulted THEN {<<p, target("D", p)>> : p \in ParentNodesOf(design)} ELSE {})
     \cup (IF kind = "unionFallback" THEN {<<u, target("D", u)>> : u \in UnionNodesOf(design)} ELSE {})
DependencyEdgesOf(rawOnly, design) ==
  {<<NodeOf("P", "raw", design), NodeOf("C", "raw", design)>>}
  \cup (IF rawOnly \/ ~JSONDistinct("P", design) THEN {}
        ELSE {<<NodeOf("P", "json", design), NodeOf("C", "json", design)>>,
              <<ChildPlanParent, NodeOf("C", "json", design)>>})

StructuralEdges == AllOfEdgesOf(ChildPlanBase, Design) \cup HintEdgesOf(RawHintRefs, Design)
Dependencies == DependencyEdgesOf(RawDependenciesOnly, Design)
PublicationEdges == StructuralEdges \cup Dependencies
IREdges == StructuralEdges \cup (IF IRFollowsDependencies THEN Dependencies ELSE {})

(* Registration arbitration: a raw registration is visited only when no JSON
   variant of its declaration is reached; unreached raw components of such a
   declaration are discarded. JSON projections are never discarded. *)
Created == Roots \cup {e[1] : e \in PublicationEdges} \cup {e[2] : e \in PublicationEdges}
          \cup {Node(x, "raw") : x \in Decls}
OccurrenceReach == Closure(Roots, IREdges)
JSONReached == {n[1] : n \in {m \in OccurrenceReach : m[2] # "raw"}}
RegistrationReach == Closure({Node(x, "raw") : x \in Decls \ JSONReached}, IREdges)
IRRetained ==
  {n \in Created : n[2] # "raw" \/ n \in OccurrenceReach \cup RegistrationReach
                   \/ n[1] \notin JSONReached}

(* Allocation quotient. Bodies of the raw and JSON representations of a
   declaration without own bytes are equal; the corrected quotient also
   compares their dependency targets, the body-only quotient does not. *)
DependencyTargets(n) == {e[2] : e \in {d \in Dependencies : d[1] = n}}
Merged(n) == IF n[2] = "json" /\ ~OwnBytes(n[1]) /\ (BodyQuotient \/ DependencyTargets(n) = DependencyTargets(<<n[1], "raw">>))
             THEN <<n[1], "raw">> ELSE n

(* Independent expectation from the corrected design and full occurrence
   resolution, whatever Mode is under check. *)
Expected == Closure(RootsOf("full", TRUE, FALSE, "json"), AllOfEdgesOf(FALSE, "full") \cup HintEdgesOf(FALSE, "full")
                                            \cup DependencyEdgesOf(FALSE, "full"))

Init ==
  /\ scenario \in Cases
  /\ phase = "arbitrate"
  /\ retainedIR = {}
  /\ reachable = {}
  /\ pending = {}

Arbitrate ==
  /\ phase = "arbitrate"
  /\ retainedIR' = IRRetained
  /\ reachable' = Roots
  /\ pending' = Roots
  /\ phase' = "walk"
  /\ UNCHANGED scenario

Visit ==
  /\ phase = "walk"
  /\ \E node \in pending:
       LET targets == {e[2] : e \in {edge \in PublicationEdges : edge[1] = node}}
       IN /\ reachable' = reachable \cup targets
          /\ pending' = (pending \ {node}) \cup (targets \ reachable)
  /\ UNCHANGED <<scenario, phase, retainedIR>>

Finish ==
  /\ phase = "walk"
  /\ pending = {}
  /\ phase' = "done"
  /\ UNCHANGED <<scenario, retainedIR, reachable, pending>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Arbitrate \/ Visit \/ Finish \/ Done
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

Retained == reachable \cap retainedIR

TypeOK ==
  /\ scenario \in Cases
  /\ phase \in {"arbitrate", "walk", "done"}
  /\ pending \subseteq reachable
CompletePublication == phase = "done" => Expected \subseteq Retained
NoExtraneousPublication == phase = "done" => Retained \subseteq Expected
NoDanglingReferences == phase = "done" => reachable \subseteq retainedIR
SameRepresentationChildren ==
  phase = "done" =>
    \A e \in AllOfEdgesOf(ChildPlanBase, Design):
      e[1] \in Retained => e[2] = Node("P", Rep(e[1]))
(* Oracle independent of the design: every byte-bearing declaration published
   through a JSON occurrence, by reference or discriminator dependency, is in
   its JSON representation. *)
JSONContextEncoding ==
  phase = "done" =>
    \A n \in Closure(JSONRoots, PublicationEdges) \cap Retained:
      OwnBytes(n[1]) => n[2] # "raw"
(* After the quotient, each component's implicit children share one
   representation. *)
ConsistentDependencies ==
  phase = "done" =>
    \A m \in {Merged(n) : n \in Retained}:
      LET targets == UNION {DependencyTargets(n) : n \in {k \in Retained : Merged(k) = m}}
      IN \A a, b \in targets: a[1] = b[1] => Rep(a) = Rep(b)
(* Oracle independent of the design and of acquisition: every published
   declaration that an occurrence attribute authors as a composition
   contributor is retained in some representation. *)
AuthoredComposition ==
  phase = "done" =>
    \A j \in 1..Len(OccurrencePath):
      \A x \in OccurrencePath[j].contributors \cap Decls: \E n \in Retained: n[1] = x
Terminates == <> (phase = "done")
=============================================================================
