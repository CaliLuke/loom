------------------------- MODULE HTTPBodyProjection -------------------------
(* Revised: composition identity of a projected HTTP body occurrence.

   Parent P declares kind (discriminator, required), mode (optional, values
   {a,b}) and id (required). Child C extends P with the allOf reference
   marker, narrows mode to {a}, requires mode and adds thumb. Sibling S
   extends P, keeps mode {a,b} and adds note. An occurrence is a body
   projection of C. Its context has a codec, members excluded by bindings, an
   occurrence-level encoding override (mode as JSON content) and, for an
   explicit inline Body, an authored subset.

   Four facts stay apart: each declaration's own contract (members, values,
   requiredness, hint); the occurrence's selection; the occurrence's wire facts
   (name and codec per authored member); and component identity. A design
   decides which P component the body references, its content, and the
   siblings P publishes. The invariants are computed from the facts. *)
EXTENDS FiniteSets, TLC

CONSTANTS Mode, Cases
VARIABLES scenario, done

vars == <<scenario, done>>

PMembers == {"kind", "mode", "id"}
Members(d) == CASE d = "P" -> PMembers [] d = "C" -> PMembers \cup {"thumb"} [] d = "S" -> PMembers \cup {"note"}
Dom(d, m) == IF m # "mode" THEN "any" ELSE IF d = "C" THEN "a" ELSE "ab"
Meet(x, y) == IF x = "any" THEN y ELSE IF y = "any" THEN x ELSE IF x = "a" \/ y = "a" THEN "a" ELSE "ab"

O(codec, excluded, override, optKind, subset) ==
  [codec |-> codec, excluded |-> excluded, override |-> override, optKind |-> optKind, subset |-> subset]
JSONFull == O("json", {}, FALSE, FALSE, FALSE)
FormOverride == O("form", {}, TRUE, FALSE, FALSE)
Occurrences(s) ==
  CASE s = "jsonFull" -> {JSONFull}
    [] s = "jsonExcludeId" -> {O("json", {"id"}, FALSE, FALSE, FALSE)}
    [] s = "formFull" -> {O("form", {}, FALSE, FALSE, FALSE)}
    [] s = "formOverride" -> {FormOverride}
    [] s = "excludeId" -> {O("form", {"id"}, FALSE, FALSE, FALSE)}
    [] s = "excludeKind" -> {O("form", {"kind"}, FALSE, FALSE, FALSE)}
    [] s = "excludeAllP" -> {O("form", PMembers, FALSE, FALSE, FALSE)}
    [] s = "optionalKind" -> {O("form", {}, FALSE, TRUE, FALSE)}
    [] s = "explicitSubset" -> {O("form", {}, FALSE, FALSE, TRUE)}
    [] s = "mixed" -> {JSONFull, FormOverride}
    [] s = "excludeThumb" -> {O("form", {"thumb"}, FALSE, FALSE, FALSE)}
    [] s = "mixedChildExclusion" -> {O("form", {}, FALSE, FALSE, FALSE), O("form", {"thumb"}, FALSE, FALSE, FALSE)}

Req(o, d) == (IF o.optKind THEN {"id"} ELSE {"kind", "id"}) \cup (IF d = "C" THEN {"mode"} ELSE {})
(* Occurrence wire facts per authored member: form names come from member tags
   (kind->k, thumb->t); the override makes mode JSON content in this context. *)
Wire(o, m) == [w |-> IF o.codec = "form" /\ m = "kind" THEN "k" ELSE IF o.codec = "form" /\ m = "thumb" THEN "t" ELSE m,
               c |-> IF o.codec = "json" \/ (o.override /\ m = "mode") THEN "json" ELSE "text"]
(* Wire facts derived from the declaration alone, ignoring the occurrence override. *)
DeclWire(o, m) == Wire([o EXCEPT !.override = FALSE], m)
W(o, m, decl) == IF decl THEN DeclWire(o, m) ELSE Wire(o, m)
Selected(o, d) == IF o.subset THEN {"thumb", "mode"} ELSE Members(d) \ o.excluded

Canonical(o) == o.codec = "json" /\ o.excluded = {} /\ ~o.override
Key(o, d) == IF Canonical(o) THEN <<d, "canonical">> ELSE <<d, o.codec, o.excluded, o.override, o.optKind>>
Props(o, d, domOwner, decl) ==
  {[m |-> m, w |-> W(o, m, decl).w, c |-> W(o, m, decl).c, dom |-> Dom(domOwner, m)] : m \in Members(d) \ o.excluded}
CanonicalP(o) ==
  LET j == [o EXCEPT !.codec = "json", !.excluded = {}, !.override = FALSE]
  IN [key |-> <<"P", "canonical">>, props |-> Props(j, "P", "P", FALSE),
      req |-> {Wire(j, m).w : m \in Req(j, "P")}, disc |-> "kind"]

(* Designs.
   current: lowering drops the named edge; a body keeps P only by reusing the
     canonical C component (JSON, nothing excluded or overridden).
   naive: restore the canonical JSON P unprojected.
   dropPartial: the prior candidate: project P only when every P member is kept.
   childDerived: the prior section 3.2: derive projected P from the child body.
   ownContract: the revised candidate: project each declaration's own contract
     through the occurrence context; partial and empty bases included; a bound
     hint follows its member's wire name or is suppressed when it is excluded.
   subsetInherits, sharedKey, declarationWire, staleHint, noSuppress:
     ownContract with one defect each. *)
Candidate == Mode \in {"ownContract", "subsetInherits", "sharedKey", "declarationWire", "staleHint", "noSuppress", "dropPartial"}
Disc(o) ==
  CASE Mode = "staleHint" -> IF "kind" \in o.excluded THEN "none" ELSE "kind"
    [] Mode = "noSuppress" -> IF "kind" \in o.excluded THEN "kind" ELSE Wire(o, "kind").w
    [] OTHER -> IF "kind" \in o.excluded THEN "none" ELSE Wire(o, "kind").w
(* The current implementation intersects the global selection with Parent's
   local members while rebuilding Parent. Both forwarding and memo identity
   then lose child-only exclusions. The two partial modes isolate each half. *)
Filtered(o) == [o EXCEPT !.excluded = o.excluded \cap PMembers]
Forwarded(o) ==
  IF Mode = "publicationOverfiltered" THEN [o EXCEPT !.excluded = Members("C") \cup Members("S")]
  ELSE IF Mode \in {"parentFiltered", "parentPropagationFiltered"} THEN Filtered(o) ELSE o
ParentIdentityContext(o) == IF Mode \in {"parentFiltered", "parentKeyFiltered"} THEN Filtered(o) ELSE o
OwnP(o) ==
  [key |-> IF Mode = "sharedKey" THEN <<"P", "canonical">> ELSE Key(ParentIdentityContext(o), "P"),
   props |-> Props(o, "P", "P", Mode = "declarationWire"),
   req |-> {W(o, m, Mode = "declarationWire").w : m \in Req(o, "P") \ o.excluded},
   disc |-> Disc(o)]
ChildDerivedP(o) ==
  [key |-> Key(o, "P"), props |-> Props(o, "P", "C", FALSE),
   req |-> {Wire(o, m).w : m \in (Req(o, "C") \cap PMembers) \ o.excluded}, disc |-> Disc(o)]
NoP == [key |-> "none"]
PComp(o) ==
  IF o.subset /\ Mode # "subsetInherits" THEN NoP
  ELSE CASE Mode = "current" -> IF Canonical(o) THEN CanonicalP(o) ELSE NoP
         [] Mode = "naive" -> CanonicalP(o)
         [] Mode = "dropPartial" -> IF o.excluded \cap PMembers = {} THEN OwnP(o) ELSE NoP
         [] Mode = "childDerived" -> ChildDerivedP(o)
         [] OTHER -> OwnP(o)
HasP(o) == PComp(o).key # "none"
(* Publication consumes the forwarded context, while the expected closure
   below is derived independently from the original occurrence facts. *)
Publishes(o) == HasP(o) /\ ~Canonical(o) /\ PComp(o).disc # "none" /\ Mode # "current" /\ Mode # "naive"
Published(o, d) ==
  LET f == Forwarded(o)
  IN IF Publishes(o)
     THEN {[key |-> Key(f, d), props |-> Props(f, d, d, Mode = "declarationWire"),
            req |-> {W(f, m, Mode = "declarationWire").w : m \in Req(f, d) \ f.excluded}, base |-> PComp(o).key]}
     ELSE {}
Siblings(o) == Published(o, "S")
Children(o) == Published(o, "C")
ActualPublication(o) ==
  {[declaration |-> c.key[1], props |-> c.props, req |-> c.req] : c \in Children(o) \cup Siblings(o)}
ExpectedPublication(o) ==
  IF Publishes(o)
  THEN {[declaration |-> d, props |-> Props(o, d, d, FALSE),
         req |-> {Wire(o, m).w : m \in Req(o, d) \ o.excluded}] : d \in {"C", "S"}}
  ELSE {}
Body(o) ==
  [props |-> {[m |-> m, w |-> Wire(o, m).w, c |-> Wire(o, m).c, dom |-> Dom("C", m)] : m \in Selected(o, "C")},
   req |-> {Wire(o, m).w : m \in Selected(o, "C") \cap Req(o, "C")}]
Context(o) == (IF HasP(o) THEN {PComp(o)} ELSE {}) \cup Siblings(o) \cup Children(o)
Rendered == UNION {{<<c.key, c>> : c \in Context(o)} : o \in Occurrences(scenario)}

Init == scenario \in Cases /\ done = FALSE
Next == (~done /\ done' = TRUE /\ UNCHANGED scenario) \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
TypeOK == scenario \in Cases /\ done \in BOOLEAN

(* Settled invariants. *)
RetainedComposition ==
  done => \A o \in Occurrences(scenario): ~o.subset => HasP(o) /\ PComp(o).key[1] = "P"
NoSubsetInheritance ==
  done => \A o \in Occurrences(scenario): o.subset => ~HasP(o)
NoReintroducedMembers ==
  done => \A o \in Occurrences(scenario): \A c \in Context(o):
            /\ {p.m : p \in c.props} \cap o.excluded = {}
            /\ c.req \subseteq {Wire(o, m).w : m \in (Members("S") \cup Members("C")) \ o.excluded}
WireConsistency ==
  done => \A o \in Occurrences(scenario): \A c \in Context(o): \A p \in c.props:
            p.w = Wire(o, p.m).w /\ p.c = Wire(o, p.m).c
ContributorOwnContract ==
  done => \A o \in Occurrences(scenario): HasP(o) =>
            /\ \A p \in PComp(o).props: p.dom = Dom("P", p.m)
            /\ {p.m : p \in PComp(o).props} = PMembers \ o.excluded
            /\ PComp(o).req = {Wire(o, m).w : m \in Req(o, "P") \ o.excluded}
SiblingAdmitsOwnValues ==
  done => \A o \in Occurrences(scenario): \A s \in Siblings(o): \A p \in s.props:
            LET inherited == {q.dom : q \in {q \in PComp(o).props : q.m = p.m}}
            IN \A d \in inherited: Meet(p.dom, d) = Dom("S", p.m)
HintRule ==
  done => \A o \in Occurrences(scenario): HasP(o) /\ ~Canonical(o) =>
            PComp(o).disc = IF "kind" \in o.excluded THEN "none" ELSE Wire(o, "kind").w
ChildRefinementRetained ==
  done => \A o \in Occurrences(scenario): "mode" \in Selected(o, "C") =>
            /\ \A p \in Body(o).props: p.m = "mode" => p.dom = "a"
            /\ Wire(o, "mode").w \in Body(o).req
DeterministicIdentity ==
  done => /\ \A a, b \in Rendered: a[1] = b[1] => a[2] = b[2]
          /\ \A a \in Rendered: \A o \in Occurrences(scenario): a[1] = <<"P", "canonical">> => a[2] = CanonicalP(o)

(* Local Parent schemas can be equal despite different required publication
   closures. A memo key must distinguish those contexts before acquisition. *)
SameKeyImpliesSamePublicationContext ==
  done => \A a, b \in Occurrences(scenario):
    (HasP(a) /\ HasP(b) /\ PComp(a).key = PComp(b).key) => ExpectedPublication(a) = ExpectedPublication(b)
PublicationOwnContract ==
  done => \A o \in Occurrences(scenario): Publishes(o) => ActualPublication(o) = ExpectedPublication(o)
PublicationReachable ==
  done => \A o \in Occurrences(scenario): Publishes(o) => Cardinality(Children(o)) = 1 /\ Cardinality(Siblings(o)) = 1

Terminates == <>done
=============================================================================
