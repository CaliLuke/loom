--------------------------- MODULE BlockedSelection ---------------------------
\* Per-occurrence blockers for unresolved references (#957) under strict,
\* selected and partial import. The importer computes closures with a worklist;
\* a declarative least fixed point is the oracle.
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANT Mode
ASSUME Mode \in {"checked", "placeholder", "originFilter", "fiveKinds", "assumeUnselected",
                 "unfilteredExamples"}

\* A and B are schemas, E is an example, U1 (schema) and U2 (example) are
\* blocked components that stand for failed reference occurrences.
Nodes == {"A", "B", "E", "U1", "U2"}
Blocked == {"U1", "U2"}
Examples == {"E", "U2"}
Ops == {"o1", "o2"}
PathOf(o) == IF o = "o1" THEN "/a" ELSE "/b"
TagOf(o) == IF o = "o1" THEN "t1" ELSE "t2"
\* An unresolved Path Item at "/c": its operations, and so their tags, are
\* unknown to the importer. hiddenT1 records what the missing document holds.
PathItems == {"none", "hiddenT1", "hiddenOther"}
Selections == {"none", "tag:t1", "prefix:/a", "prefix:/c"}
BlockerOf(u) == "blk:" \o u
Order == <<"A", "B", "E", "U1", "U2">>

VARIABLES edges, roots, secFail, pathItem, sel,
          phase, selected, pathSelected, work, closure, retained, rendered,
          pending, opWork, opClosure, written, skippedPaths
inputs == <<edges, roots, secFail, pathItem, sel>>
vars == <<edges, roots, secFail, pathItem, sel, phase, selected, pathSelected,
          work, closure, retained, rendered, pending, opWork, opClosure, written,
          skippedPaths>>

Matches(o) == CASE sel = "none" -> TRUE
                [] sel = "tag:t1" -> TagOf(o) = "t1"
                [] sel = "prefix:/a" -> PathOf(o) = "/a"
                [] sel = "prefix:/c" -> FALSE
\* Tags of an unresolved Path Item are unknowable; a tag selector must keep it.
PathItemMatches == CASE sel = "none" -> TRUE
                     [] sel = "tag:t1" -> Mode # "assumeUnselected"
                     [] sel = "prefix:/a" -> FALSE
                     [] sel = "prefix:/c" -> TRUE
\* fiveKinds is the current closure: it does not traverse example components.
Followed(n) == Mode # "fiveKinds" \/ n \notin Examples
Succ(n) == {m \in edges[n] : Followed(m)}
RootsOf(S) == {n \in UNION {roots[o] : o \in S} : Followed(n)}
First(S) == Order[CHOOSE i \in 1..5 : Order[i] \in S /\ \A j \in 1..(i - 1) : Order[j] \notin S]
Recorded == (IF Mode = "placeholder" THEN {} ELSE {BlockerOf(u) : u \in Blocked})
            \cup (IF secFail THEN {"sec:o2"} ELSE {})
            \cup (IF pathItem # "none" THEN {"path:/c"} ELSE {})
\* An origin-keyed filter looks up the source component name, not the bundled key.
OriginKey(u) == "external-name"
Keep(d) == CASE d = "sec:o2" -> "o2" \in selected
             [] d = "path:/c" -> pathSelected
             [] OTHER -> LET u == IF d = "blk:U1" THEN "U1" ELSE "U2"
                         IN CASE Mode = "originFilter" -> OriginKey(u) \in closure
                              \* Current filtering never drops example-component diagnostics.
                              [] Mode = "unfilteredExamples" /\ u \in Examples -> TRUE
                              [] OTHER -> u \in closure
UnfilteredExample(v) == Mode = "unfilteredExamples" /\ v \in Examples
OpBlocked(c) == {v \in Blocked : v \in c \/ UnfilteredExample(v)}
OpDiagnostics(o, c) == ({BlockerOf(u) : u \in OpBlocked(c)} \cap Recorded)
                       \cup (IF o = "o2" /\ secFail THEN {"sec:o2"} ELSE {})

Init ==
    /\ edges \in {[n \in Nodes |-> CASE n = "A" -> a [] n = "B" -> b [] n = "E" -> e [] OTHER -> {}] :
                   a \in SUBSET {"B", "U1", "E"}, b \in SUBSET {"A", "U1"}, e \in SUBSET {"U2"}}
    /\ roots \in {[o \in Ops |-> IF o = "o1" THEN r1 ELSE r2] :
                   r1 \in SUBSET {"A", "E"}, r2 \in SUBSET {"B", "E"}}
    /\ secFail \in BOOLEAN
    /\ pathItem \in PathItems
    /\ sel \in Selections
    /\ phase = "select"
    /\ selected = {} /\ pathSelected = FALSE /\ work = {} /\ closure = {}
    /\ retained = {} /\ rendered = FALSE /\ pending = <<>> /\ opWork = {}
    /\ opClosure = {} /\ written = {} /\ skippedPaths = {}

Select ==
    /\ phase = "select"
    /\ selected' = {o \in Ops : Matches(o)}
    /\ pathSelected' = (pathItem # "none" /\ PathItemMatches)
    \* Strict import keeps every component; selection starts from operation roots.
    /\ IF sel = "none" THEN closure' = Nodes /\ work' = {} /\ phase' = "filter"
       ELSE closure' = {} /\ work' = RootsOf({o \in Ops : Matches(o)}) /\ phase' = "closure"
    /\ UNCHANGED <<inputs, retained, rendered, pending, opWork, opClosure, written, skippedPaths>>

Visit ==
    /\ phase = "closure" /\ work # {}
    /\ LET n == First(work)
       IN /\ closure' = closure \cup {n}
          /\ work' = (work \ {n}) \cup (Succ(n) \ (closure \cup {n}))
    /\ UNCHANGED <<inputs, phase, selected, pathSelected, retained, rendered, pending,
                   opWork, opClosure, written, skippedPaths>>

ClosureDone ==
    /\ phase = "closure" /\ work = {}
    /\ phase' = "filter"
    /\ UNCHANGED <<inputs, selected, pathSelected, work, closure, retained, rendered,
                   pending, opWork, opClosure, written, skippedPaths>>

\* Diagnostics are retained by bundled path; the shared planner renders only
\* when no blocker is retained.
Filter ==
    /\ phase = "filter"
    \* Strict import does not filter: every recorded blocker is retained.
    /\ LET kept == IF sel = "none" THEN Recorded ELSE {d \in Recorded : Keep(d)}
       IN retained' = kept /\ rendered' = (kept = {})
    /\ pending' = <<"o1", "o2">>
    /\ phase' = "partial"
    /\ skippedPaths' = IF pathSelected THEN {"/c"} ELSE {}
    /\ UNCHANGED <<inputs, selected, pathSelected, work, closure, opWork, opClosure, written>>

\* Partial import computes one closure per selected operation.
PartialStart ==
    /\ phase = "partial" /\ pending # <<>>
    /\ IF Head(pending) \in selected
       THEN /\ opWork' = RootsOf({Head(pending)}) /\ opClosure' = {}
            /\ phase' = "opClosure" /\ UNCHANGED pending
       ELSE /\ pending' = Tail(pending) /\ UNCHANGED <<opWork, opClosure, phase>>
    /\ UNCHANGED <<inputs, selected, pathSelected, work, closure, retained, rendered, written, skippedPaths>>

OpVisit ==
    /\ phase = "opClosure" /\ opWork # {}
    /\ LET n == First(opWork)
       IN /\ opClosure' = opClosure \cup {n}
          /\ opWork' = (opWork \ {n}) \cup (Succ(n) \ (opClosure \cup {n}))
    /\ UNCHANGED <<inputs, phase, selected, pathSelected, work, closure, retained, rendered,
                   pending, written, skippedPaths>>

OpDecide ==
    /\ phase = "opClosure" /\ opWork = {}
    /\ LET o == Head(pending)
       IN written' = IF OpDiagnostics(o, opClosure) = {} THEN written \cup {o} ELSE written
    /\ pending' = Tail(pending)
    /\ phase' = "partial"
    /\ UNCHANGED <<inputs, selected, pathSelected, work, closure, retained, rendered,
                   opWork, opClosure, skippedPaths>>

Finish ==
    /\ phase = "partial" /\ pending = <<>>
    /\ phase' = "done"
    /\ UNCHANGED <<inputs, selected, pathSelected, work, closure, retained, rendered,
                   pending, opWork, opClosure, written, skippedPaths>>

Terminated == phase = "done" /\ UNCHANGED vars

Next == Select \/ Visit \/ ClosureDone \/ Filter \/ PartialStart \/ OpVisit \/ OpDecide
        \/ Finish \/ Terminated
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

\* Oracle: complete reachability over every component kind.
Step(S) == S \cup UNION {edges[n] : n \in S}
RECURSIVE Lfp(_)
Lfp(S) == IF Step(S) = S THEN S ELSE Lfp(Step(S))
ReachOf(S) == Lfp(UNION {roots[o] : o \in S})
TrueSelected == {o \in Ops : Matches(o)}
Imported == IF sel = "none" THEN Nodes ELSE ReachOf(TrueSelected)
PathItemTrulySelected ==
    pathItem # "none" /\ (sel \in {"none", "prefix:/c"} \/ (sel = "tag:t1" /\ pathItem = "hiddenT1"))
Filtered == phase \in {"partial", "opClosure", "done"}

BlockerKeptIffImported ==
    Filtered => \A u \in Blocked : (BlockerOf(u) \in retained) <=> (u \in Imported)
SecurityKeptIffSelected == Filtered => (("sec:o2" \in retained) <=> (secFail /\ "o2" \in TrueSelected))
PathItemSound == Filtered => (PathItemTrulySelected => "path:/c" \in retained)
PathItemPathPrecise == Filtered => (sel = "prefix:/a" => "path:/c" \notin retained)
NoBlockedRendered == Filtered => (rendered => closure \cap Blocked = {})
PartialSound ==
    phase = "done" => \A o \in written : ReachOf({o}) \cap Blocked = {} /\ ~(o = "o2" /\ secFail)
PartialMaximal ==
    phase = "done" => \A o \in TrueSelected :
        (ReachOf({o}) \cap Blocked = {} /\ ~(o = "o2" /\ secFail)) => o \in written
PartialListsPathItem == phase = "done" => (PathItemTrulySelected => "/c" \in skippedPaths)
Termination == <>(phase = "done")
=============================================================================
