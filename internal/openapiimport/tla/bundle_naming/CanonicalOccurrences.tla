----------------------- MODULE CanonicalOccurrences -----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
VARIABLES pending, bodies, references, origins
Targets == {"A", "B", "Child", "Alias"}
Inline == {"inlineA", "inlineB", "response"}
Declarations == {"declA", "declB", "declChild", "declAlias"}
ReferenceSites == {"ordinary", "mapping", "aliasMapping"}
Tasks == Inline \cup Declarations \cup ReferenceSites
Source(site) == CASE site \in {"inlineA", "declA", "ordinary"} -> "A"
                 [] site \in {"inlineB", "declB"} -> "B"
                 [] site \in {"response", "declChild", "mapping"} -> "Child"
                 [] OTHER -> "Alias"
Canonical(target) == IF Policy = "shapeMerge" /\ target = "B"
                     THEN "declA" ELSE "decl" \o target
vars == <<pending, bodies, references, origins>>
Init == /\ pending = Tasks
        /\ bodies = [s \in Tasks |-> ""]
        /\ references = [s \in Tasks |-> ""]
        /\ origins = [s \in Tasks |-> ""]
Copy == /\ pending # {}
        /\ \E site \in pending:
             LET target == Source(site)
                 destination == Canonical(target)
                 reference == site \in ReferenceSites
                              \/ Policy = "replaceAll"
                              \/ (Policy # "duplicate" /\ site # destination)
             IN /\ pending' = pending \ {site}
                /\ bodies' = [bodies EXCEPT ![site] = IF reference THEN "" ELSE target]
                /\ references' = [references EXCEPT ![site] = IF reference THEN destination ELSE ""]
                /\ origins' = [origins EXCEPT ![site] = target]
Next == Copy \/ (pending = {} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
CanonicalUse == pending = {} => \A site \in Inline: references[site] = Canonical(Source(site))
Materialized == pending = {} => \A target \in Targets: bodies[Canonical(target)] = target
SingleDefinition == pending = {} => \A target \in Targets:
                    Cardinality({site \in Tasks: bodies[site] = target}) = 1
DistinctIdentity == pending = {} => \A a,b \in Targets: a # b => Canonical(a) # Canonical(b)
AliasesPreserved == pending = {} => references["aliasMapping"] = "declAlias"
OriginsPreserved == \A site \in Tasks \ pending: origins[site] = Source(site)
=============================================================================
