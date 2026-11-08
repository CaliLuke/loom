----------------------- MODULE QueryPublication -----------------------
EXTENDS Naturals, FiniteSets

CONSTANT Mode
VARIABLES declaring, recursive, sharedBody, phase, pending, visited, declarations,
          pendingHelpers, helpers, inputValid, stages, validated, unsafe
vars == <<declaring, recursive, sharedBody, phase, pending, visited, declarations,
          pendingHelpers, helpers, inputValid, stages, validated, unsafe>>

Consumers == {"server", "cli"}
Types == {"Choice", "Cat", "Dog"}
\* A non-declaring carrier, such as a JSON map of scalars, has a physical
\* carrier but no named declarations; "Value" stands for its whole graph.
Root == IF declaring THEN "Choice" ELSE "Value"
Declarable(d) == d.typ # "Value"
D(package, role, typ) == [package |-> package, role |-> role, typ |-> typ]
SourceEdges(typ) == CASE typ = "Choice" -> {"Cat", "Dog"}
                     [] typ = "Cat" /\ recursive -> {"Choice"}
                     [] OTHER -> {}
Edges(d) == {D(d.package, d.role, t): t \in SourceEdges(d.typ)}
RECURSIVE Expand(_, _)
Expand(roots, rounds) == IF rounds = 0 THEN roots
                        ELSE Expand(roots \cup UNION {Edges(d): d \in roots}, rounds - 1)
Closure(roots) == Expand(roots, Cardinality(Types))

Native == D("service", "native", Root)
Wire(package) == D(package, "wire", Root)
Local(package) == D(package, "local-native", Root)
Body == IF sharedBody THEN {D("server", "body", Root)} ELSE {}
OldTarget == Mode \in {"current", "registry-only"}
H(source, target) == [source |-> source, target |-> target]
PlanRoots == IF OldTarget
             THEN {H(Wire("server"), Local("server")),
                   H(Local("server"), Native), H(Local("cli"), Native)}
             ELSE {H(Wire(p), Native): p \in Consumers}
HelperEdges(h) == {H(D(h.source.package, h.source.role, t),
                    D(h.target.package, h.target.role, t)):
                  t \in SourceEdges(h.source.typ)}
RECURSIVE ExpandHelpers(_, _)
ExpandHelpers(roots, rounds) ==
    IF rounds = 0 THEN roots
    ELSE ExpandHelpers(roots \cup UNION {HelperEdges(h): h \in roots}, rounds - 1)
NeededHelpers == ExpandHelpers(PlanRoots, Cardinality(Types))
NeededDeclarations == {d \in Closure({Native} \cup Body \cup
                                    {h.source: h \in PlanRoots} \cup
                                    {h.target: h \in PlanRoots}): Declarable(d)}

PreparedRoots ==
    {Native} \cup Body \cup
    CASE Mode = "current" -> {}
      [] Mode = "registry-only" -> {Wire("server")}
      [] OTHER -> {Wire(p): p \in Consumers}
CapturedPlans ==
    IF OldTarget THEN {H(Wire("server"), Local("server"))}
    ELSE IF Mode = "dropped-helpers" THEN {H(Wire("server"), Native)}
    ELSE PlanRoots

\* Actual NameScope allocation is an assumption. These concrete names model
\* detached body/CLI identities and the old same-package native/wire collision.
Symbol(d) == <<d.package,
              IF d.role = "body" THEN <<"body", d.typ>>
              ELSE <<"ordinary", d.typ>>>>
Layout(d) == IF d.role \in {"native", "local-native"} THEN "native" ELSE "json"

Init == /\ declaring \in BOOLEAN
        /\ recursive \in BOOLEAN
        /\ sharedBody \in BOOLEAN
        /\ phase = "collect-types"
        /\ pending = PreparedRoots
        /\ visited = {}
        /\ declarations = {}
        /\ pendingHelpers = {}
        /\ helpers = {}
        /\ inputValid = [p \in Consumers |-> TRUE]
        /\ stages = [p \in Consumers |-> "waiting"]
        /\ validated = {}
        /\ unsafe = {}

AlreadyVisited(d) == IF Mode = "global-membership"
                     THEN \E seen \in visited: seen.role = d.role /\ seen.typ = d.typ
                     ELSE d \in visited
CollectType(d) ==
    /\ phase = "collect-types"
    /\ d \in pending
    /\ pending' = (pending \ {d}) \cup (IF AlreadyVisited(d) THEN {} ELSE Edges(d))
    /\ visited' = visited \cup {d}
    /\ declarations' = declarations \cup
                       (IF AlreadyVisited(d) \/ ~Declarable(d) THEN {} ELSE {d})
    /\ UNCHANGED <<declaring, recursive, sharedBody, phase, pendingHelpers, helpers,
                   inputValid, stages, validated, unsafe>>
StartHelpers ==
    /\ phase = "collect-types"
    /\ pending = {}
    /\ phase' = "collect-helpers"
    /\ pendingHelpers' = CapturedPlans
    /\ UNCHANGED <<declaring, recursive, sharedBody, pending, visited, declarations,
                   helpers, inputValid, stages, validated, unsafe>>
CollectHelper(h) ==
    /\ phase = "collect-helpers"
    /\ h \in pendingHelpers
    /\ pendingHelpers' = (pendingHelpers \ {h}) \cup
                          (IF h \in helpers THEN {} ELSE HelperEdges(h))
    /\ helpers' = helpers \cup {h}
    /\ UNCHANGED <<declaring, recursive, sharedBody, phase, pending, visited, declarations,
                   inputValid, stages, validated, unsafe>>
Publish ==
    /\ phase = "collect-helpers"
    /\ pendingHelpers = {}
    /\ phase' = "published"
    /\ UNCHANGED <<declaring, recursive, sharedBody, pending, visited, declarations,
                   pendingHelpers, helpers, inputValid, stages, validated, unsafe>>
\* The R2 CLI decoded a non-declaring JSON query flag directly into native
\* storage: no wire carrier, so no validation precedes the assignment.
NativeInput(p) == Mode = "native-cli" /\ p = "cli" /\ ~declaring
Decode ==
    /\ phase = "published"
    /\ phase' = "runtime"
    /\ inputValid' \in [Consumers -> BOOLEAN]
    /\ stages' = [p \in Consumers |->
                    IF Mode = "late-validation" \/ NativeInput(p) THEN "convert" ELSE "validate"]
    /\ UNCHANGED <<declaring, recursive, sharedBody, pending, visited, declarations,
                   pendingHelpers, helpers, validated, unsafe>>
Validate(p) ==
    /\ phase = "runtime"
    /\ p \in Consumers
    /\ stages[p] = "validate"
    /\ validated' = validated \cup {p}
    /\ stages' = [stages EXCEPT ![p] = IF inputValid[p] THEN "convert" ELSE "rejected"]
    /\ UNCHANGED <<declaring, recursive, sharedBody, phase, pending, visited, declarations,
                   pendingHelpers, helpers, inputValid, unsafe>>
Convert(p) ==
    /\ phase = "runtime"
    /\ p \in Consumers
    /\ stages[p] = "convert"
    /\ unsafe' = unsafe \cup (IF p \in validated /\ inputValid[p] THEN {} ELSE {p})
    /\ stages' = [stages EXCEPT ![p] = "done"]
    /\ UNCHANGED <<declaring, recursive, sharedBody, phase, pending, visited, declarations,
                   pendingHelpers, helpers, inputValid, validated>>

Next == (\E d \in pending: CollectType(d)) \/ StartHelpers \/
        (\E h \in pendingHelpers: CollectHelper(h)) \/ Publish \/ Decode \/
        (\E p \in Consumers: Validate(p) \/ Convert(p))
Spec == Init /\ [][Next]_vars
Published == phase \in {"published", "runtime"}
EveryReferenceDeclared == Published => NeededDeclarations \subseteq declarations
OneLayoutPerSymbol == Published =>
    \A a, b \in NeededDeclarations: Symbol(a) = Symbol(b) => Layout(a) = Layout(b)
EveryHelperPublished == Published => NeededHelpers \subseteq helpers
NoUnneededDeclarations == declarations \subseteq NeededDeclarations
ValidateBeforeConvert == unsafe = {}
CorrectResult == phase = "runtime" => \A p \in Consumers:
    (stages[p] = "done" => inputValid[p]) /\
    (stages[p] = "rejected" => ~inputValid[p])
=============================================================================
