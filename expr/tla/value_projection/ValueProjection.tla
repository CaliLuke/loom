----------------------- MODULE ValueProjection -----------------------
EXTENDS Naturals, Sequences, FiniteSets

(***************************************************************************
Two consumers interleave source selection, resolution, projection and target
validation. A branch path represents two nested unions; payload shape is
intentionally identical for all paths. Scalar encoding is outside this model.

Mode selects an isolated legacy/rejected transition. Scenario limits a fault
witness; "all" checks the complete finite scenario set. See README for the
current Go seams and the distinction between a bug and a negative control.
***************************************************************************)
CONSTANTS Mode, Scenario
Jobs == {1, 2}
Paths == {<<"a", "x">>, <<"a", "y">>, <<"b", "x">>, <<"b", "y">>}
Presence == {"absent", "null", "present"}
Roles == {"example", "enum", "default"}
Reps == {"raw", "resolved", "wire"}
Scenarios == {"ordinary", "occurrence", "source", "role", "target",
              "synthesis", "missing-contract", "suppressed",
              "excluded", "unrepresentable"}
Phases == {"new", "selected", "resolved", "projected", "validated", "done"}
Modes == {"checked", "legacy", "reselection", "resynthesize", "precedence", "always-omit",
          "cache-occurrence", "cache-source", "cache-role", "cache-phase", "cache-target"}
ASSUME /\ Mode \in Modes
       /\ Scenario \in Scenarios \cup {"all"}

VARIABLES scenario, choice, inputPresence, contractRole,
          phase, source, snapshot, cache, outcome, events
vars == <<scenario, choice, inputPresence, contractRole,
          phase, source, snapshot, cache, outcome, events>>

Occurrence(j) == IF scenario = "occurrence" THEN j ELSE 1
Role(j) == IF scenario \in {"role", "unrepresentable", "suppressed"} /\ j = 2 THEN contractRole
           ELSE IF scenario = "missing-contract" THEN contractRole
           ELSE "example"
Target(j) == IF scenario = "target" /\ j = 2 THEN "body" ELSE "full"

\* Source tiers stand for ExtractUserExamples' local then inherited precedence.
\* Their payloads are abstract; the source identity is not a payload hash.
HasLocal(j) == scenario \notin {"synthesis", "missing-contract"}
               /\ ~(scenario = "source" /\ j = 1)
HasInherited(j) == scenario \notin {"synthesis", "missing-contract"}
SelectedSource(j) == IF HasLocal(j) THEN "local"
                     ELSE IF HasInherited(j) THEN "inherited"
                     ELSE "none"
Eligible(j) == scenario # "excluded"
               /\ ~(scenario = "suppressed" /\ Role(j) = "example")
ExpectedSource(j) == IF SelectedSource(j) = "none" /\ Role(j) = "example"
                    THEN "synthetic" ELSE SelectedSource(j)
Authored(j) == SelectedSource(j) # "none"

\* A selected body/view may discard the inner field legitimately. This is an
\* observation plan, specified independently of Project below, not reselection.
ObservedPath(j) == IF Target(j) = "body" THEN <<choice[j][1]>> ELSE choice[j]
ObservedPresence(j) == IF Target(j) = "body" THEN "absent" ELSE inputPresence[j]

Owner(j, rep) == [occurrence |-> Occurrence(j), source |-> ExpectedSource(j),
                  role |-> Role(j), rep |-> rep,
                  target |-> IF rep = "wire" THEN Target(j) ELSE "none"]
Key(owner) == [occurrence |-> IF Mode = "cache-occurrence" THEN 0 ELSE owner.occurrence,
               source |-> IF Mode = "cache-source" THEN "shared" ELSE owner.source,
               role |-> IF Mode = "cache-role" THEN "shared" ELSE owner.role,
               rep |-> IF Mode = "cache-phase" THEN "shared" ELSE owner.rep,
               target |-> IF Mode = "cache-target" THEN "shared" ELSE owner.target]
Entry(j, rep, path, presence, origin) ==
    [owner |-> Owner(j, rep), path |-> path, presence |-> presence, origin |-> origin]

\* Cache entries are immutable owned records. There is at most one per key.
Cached(key) == {e \in cache : Key(e.owner) = key}
ReadOrNew(entry) == IF Cached(Key(entry.owner)) = {} THEN entry
                   ELSE CHOOSE e \in Cached(Key(entry.owner)) : TRUE
Add(entry) == IF Cached(Key(entry.owner)) = {} THEN cache \cup {entry} ELSE cache

Init == /\ scenario \in IF Scenario = "all" THEN Scenarios ELSE {Scenario}
        /\ choice \in [Jobs -> Paths]
        /\ inputPresence \in [Jobs -> Presence]
        /\ contractRole \in {"enum", "default"}
        /\ \A j, k \in Jobs : Owner(j, "resolved") = Owner(k, "resolved")
            => /\ choice[j] = choice[k]
               /\ inputPresence[j] = inputPresence[k]
        /\ phase = [j \in Jobs |-> "new"]
        /\ source = [j \in Jobs |-> "none"]
        /\ snapshot = [j \in Jobs |->
            [owner |-> [occurrence |-> 0, source |-> "none", role |-> "example",
                       rep |-> "none", target |-> "none"],
             path |-> <<>>, presence |-> "absent", origin |-> "none"]]
        /\ cache = {}
        /\ outcome = [j \in Jobs |-> "pending"]
        /\ events = [j \in Jobs |-> <<>>]

Skip(j) == /\ phase[j] = "new"
           /\ ~Eligible(j) \/ (SelectedSource(j) = "none" /\ Role(j) # "example")
           /\ phase' = [phase EXCEPT ![j] = "done"]
           /\ outcome' = [outcome EXCEPT ![j] = IF ~Eligible(j) THEN "suppressed" ELSE "no-value"]
           /\ events' = [events EXCEPT ![j] = Append(@, "skip")]
           /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, source, snapshot, cache>>

Select(j) ==
    LET entry == Entry(j, "raw", <<>>, inputPresence[j], ExpectedSource(j))
    IN /\ phase[j] = "new"
       /\ Eligible(j)
       /\ SelectedSource(j) # "none" \/ Role(j) = "example"
       /\ source' = [source EXCEPT ![j] = IF Mode = "precedence" /\ HasLocal(j)
                                         THEN "inherited" ELSE ExpectedSource(j)]
       /\ snapshot' = [snapshot EXCEPT ![j] = ReadOrNew(entry)]
       /\ cache' = Add(entry)
       /\ phase' = [phase EXCEPT ![j] = "selected"]
       /\ events' = [events EXCEPT ![j] = Append(@, "select")]
       /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, outcome>>

Resolve(j) ==
    LET path == IF Mode = "legacy" THEN <<choice[j][2]>> ELSE choice[j]
        origin == IF Mode = "resynthesize" /\ Authored(j) THEN "synthetic" ELSE source[j]
        entry == Entry(j, "resolved", path, snapshot[j].presence, origin)
    IN /\ phase[j] = "selected"
       /\ snapshot' = [snapshot EXCEPT ![j] = ReadOrNew(entry)]
       /\ cache' = Add(entry)
       /\ phase' = [phase EXCEPT ![j] = "resolved"]
       /\ events' = [events EXCEPT ![j] = Append(@, "resolve")]
       /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, source, outcome>>

Project(j) ==
    \* Payload-only inference cannot distinguish the two outer branches. The
    \* rejected repair picks "a". Checked projection reads the retained path.
    LET retained == IF Mode = "reselection" THEN <<"a", snapshot[j].path[2]>>
                    ELSE snapshot[j].path
        path == IF Target(j) = "body" THEN <<retained[1]>> ELSE retained
        presence == IF Target(j) = "body" THEN "absent" ELSE snapshot[j].presence
        entry == Entry(j, "wire", path, presence, snapshot[j].origin)
    IN /\ phase[j] = "resolved"
       /\ snapshot' = [snapshot EXCEPT ![j] = ReadOrNew(entry)]
       /\ cache' = Add(entry)
       /\ phase' = [phase EXCEPT ![j] = "projected"]
       /\ events' = [events EXCEPT ![j] = Append(@, "project")]
       /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, source, outcome>>

Validate(j) ==
    /\ phase[j] = "projected"
    /\ phase' = [phase EXCEPT ![j] = "validated"]
    /\ outcome' = [outcome EXCEPT ![j] = IF scenario = "unrepresentable"
                                            THEN IF Role(j) = "example" THEN "omitted" ELSE "error"
                                            ELSE IF Mode = "always-omit" THEN "omitted" ELSE "ready"]
    /\ events' = [events EXCEPT ![j] = Append(@, "validate")]
    /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, source, snapshot, cache>>

Finish(j) == /\ phase[j] = "validated"
             /\ phase' = [phase EXCEPT ![j] = "done"]
             /\ outcome' = [outcome EXCEPT ![j] = IF @ = "ready" THEN "emitted" ELSE @]
             /\ events' = [events EXCEPT ![j] = Append(@, "finish")]
             /\ UNCHANGED <<scenario, choice, inputPresence, contractRole, source, snapshot, cache>>

Step(j) == Skip(j) \/ Select(j) \/ Resolve(j) \/ Project(j) \/ Validate(j) \/ Finish(j)
Done == \A j \in Jobs : phase[j] = "done"
Next == (\E j \in Jobs : Step(j)) \/ (Done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

TypeOK == /\ scenario \in Scenarios
          /\ choice \in [Jobs -> Paths]
          /\ inputPresence \in [Jobs -> Presence]
          /\ phase \in [Jobs -> Phases]
          /\ source \in [Jobs -> {"none", "local", "inherited", "synthetic"}]

Active(j) == snapshot[j].owner.rep # "none"
Representation(j) == CASE phase[j] = "selected" -> "raw"
                       [] phase[j] = "resolved" -> "resolved"
                       [] OTHER -> "wire"
CacheOwnership == \A j \in Jobs : Active(j) => snapshot[j].owner = Owner(j, Representation(j))
SourcePrecedence == \A j \in Jobs : Active(j) => source[j] = ExpectedSource(j)
NoAuthoredResynthesis == \A j \in Jobs : Active(j) /\ Authored(j) => snapshot[j].origin = SelectedSource(j)
ContractNeverSynthesized == \A j \in Jobs : Role(j) # "example" => source[j] # "synthetic"
BranchRetention == \A j \in Jobs : phase[j] = "resolved" => snapshot[j].path = choice[j]
ProjectionRetention == \A j \in Jobs : phase[j] \in {"projected", "validated", "done"} /\ Active(j)
                       => /\ snapshot[j].path = ObservedPath(j)
                          /\ snapshot[j].presence = ObservedPresence(j)
PresenceRetention == \A j \in Jobs : phase[j] \in {"selected", "resolved"}
                     => snapshot[j].presence = inputPresence[j]
SuppressionBeforeSelection == \A j \in Jobs : ~Eligible(j) => ~Active(j)

\* Each consumer advances monotonically; all interleavings remain possible.
History(j) == events[j]
PhaseOrder == \A j \in Jobs :
    LET h == History(j)
        steps == <<"select", "resolve", "project", "validate", "finish">>
    IN h = <<>> \/ h = <<"skip">> \/
       (Len(h) <= Len(steps) /\ \A n \in 1..Len(h) : h[n] = steps[n])
\* This is an independent terminal specification, including reachable omission
\* and contract-error cases. Always omitting or silently skipping fails it.
ExpectedOutcome(j) == IF ~Eligible(j) THEN "suppressed"
                      ELSE IF SelectedSource(j) = "none" /\ Role(j) # "example" THEN "no-value"
                      ELSE IF scenario = "unrepresentable"
                           THEN IF Role(j) = "example" THEN "omitted" ELSE "error"
                      ELSE "emitted"
TerminalOutcome == \A j \in Jobs : phase[j] = "done" => outcome[j] = ExpectedOutcome(j)
EmissionAfterValidation == \A j \in Jobs : outcome[j] = "emitted"
                           => /\ phase[j] = "done"
                              /\ History(j) = <<"select", "resolve", "project", "validate", "finish">>
Termination == <>Done
=============================================================================
