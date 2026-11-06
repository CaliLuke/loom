---- MODULE ObjectComposition ----
\* Bounded model of Loom Extend over objects with additional-member policies.
\*
\* Semantics (Spec): every source constrains every key it does not itself
\* declare (JSON Schema allOf with subschema-local additionalProperties).
\* Roots are the authored bases. Each chain level is a DSL Extend of all
\* earlier sources; it semantically declares every inherited declaration
\* (contributing no constraint) plus its local declarations.
\*
\* A source's req may name keys it does not declare. Such keys keep their
\* semantic classification; Lowering decides only the physical layout.
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANTS RootCount, ChainLength, RootKeys, LocalKeys, RootReqKeys, LocalReqKeys,
          DeclSchemas, TypedSchemas, PolicyKinds, Lowering

Keys == {"id", "note", "x"}
Values == {"zero", "neg", "str", "null"}
Absent == "absent"

SchemaNames == {"int", "int0", "str", "nint", "nstr"}
Schema(n) == CASE n = "int" -> {"zero", "neg"}
             [] n = "int0" -> {"zero"}
             [] n = "str" -> {"str"}
             [] n = "nint" -> {"zero", "neg", "null"}
             [] n = "nstr" -> {"str", "null"}
\* Domains Loom can carry as one field or one extras value type.
Representable(S) == S = Values \/ \E n \in SchemaNames : Schema(n) = S

Open == [kind |-> "open", schema |-> Values]
Closed == [kind |-> "closed", schema |-> {}]
Policies == {p \in {Open, Closed} \cup {[kind |-> "typed", schema |-> Schema(n)] : n \in TypedSchemas}
                : p.kind \in PolicyKinds}

Decls(K) == UNION {[D -> {Schema(n) : n \in DeclSchemas}] : D \in SUBSET K}
RootSources == [decl : Decls(RootKeys), policy : Policies, req : SUBSET RootReqKeys]
LocalSources == [decl : Decls(LocalKeys), policy : Policies, req : SUBSET LocalReqKeys]
RootSeqs == IF RootCount = 1 THEN {<<r>> : r \in RootSources}
            ELSE {<<r, q>> : r, q \in RootSources}
ChainSeqs == IF ChainLength = 1 THEN {<<l>> : l \in LocalSources}
             ELSE {<<l, m>> : l, m \in LocalSources}
Designs == [roots : RootSeqs, chain : ChainSeqs]
Instances == [Keys -> Values \cup {Absent}]

Inter(S) == {v \in Values : \A X \in S : v \in X}

RootSet(d) == {d.roots[i] : i \in DOMAIN d.roots}
LevelSource(d, i) ==
    LET inherited == UNION {DOMAIN r.decl : r \in RootSet(d)}
                     \cup UNION {DOMAIN d.chain[j].decl : j \in 1..(i - 1)}
        local == d.chain[i].decl
    IN [decl |-> [k \in inherited \cup DOMAIN local |->
                    IF k \in DOMAIN local THEN local[k] ELSE Values],
        policy |-> d.chain[i].policy,
        req |-> d.chain[i].req]
Sources(d) == RootSet(d) \cup {LevelSource(d, i) : i \in DOMAIN d.chain}

SourceAccepts(s, inst) ==
    /\ \A k \in s.req : inst[k] # Absent
    /\ \A k \in Keys : inst[k] # Absent =>
         IF k \in DOMAIN s.decl THEN inst[k] \in s.decl[k] ELSE inst[k] \in s.policy.schema
Spec(d, inst) == \A s \in Sources(d) : SourceAccepts(s, inst)

\* Rejected alternative: rewrite a typed root's required undeclared keys into
\* semantic declarations before composition. Later levels then inherit them.
Promote(d) ==
    [d EXCEPT !.roots = [i \in DOMAIN d.roots |->
        LET r == d.roots[i] IN
        IF r.policy.kind = "typed"
        THEN [r EXCEPT !.decl = [k \in DOMAIN r.decl \cup r.req |->
                                   IF k \in DOMAIN r.decl THEN r.decl[k] ELSE r.policy.schema]]
        ELSE r]]
Lowered(d) == IF Lowering = "semantic-promotion" THEN Promote(d) ELSE d

\* Physical layout: declared fields plus one map for all undeclared keys.
\* Required keys remain semantic obligations, independent of their storage.
\* Every field and the map values carry the per-source conjunction.
Compose(d) ==
    LET semantic == UNION {DOMAIN s.decl : s \in Sources(d)}
        required == UNION {s.req : s \in Sources(d)}
        physical == semantic \cup required
        obligation == [k \in physical |->
            Inter({IF k \in DOMAIN s.decl THEN s.decl[k] ELSE s.policy.schema : s \in Sources(d)})]
        extras == Inter({s.policy.schema : s \in Sources(d)})
        unsat == \E k \in required : obligation[k] = {}
        unsupported ==
            /\ ~unsat
            /\ \/ \E k \in physical : obligation[k] # {} /\ ~Representable(obligation[k])
               \/ \E k \in physical \ required : obligation[k] = {}
               \/ extras # {} /\ ~Representable(extras)
    IN [fields |-> [k \in semantic |-> obligation[k]], extras |-> extras, req |-> required,
        requiredExtras |-> required \ semantic,
        outcome |-> IF unsat THEN "unsatisfiable"
                    ELSE IF unsupported THEN "unsupported" ELSE "admitted"]

LoomAccepts(c, inst) ==
    /\ \A k \in c.req : inst[k] # Absent
    /\ \A k \in Keys : inst[k] # Absent =>
         IF k \in DOMAIN c.fields THEN inst[k] \in c.fields[k] ELSE inst[k] \in c.extras

VARIABLE design
Init == design \in Designs
Next == UNCHANGED design

Sound ==
    LET c == Compose(Lowered(design)) IN
    c.outcome = "admitted" => \A inst \in Instances : LoomAccepts(c, inst) = Spec(design, inst)

\* "Unsatisfiable" is reserved for contracts with no valid document.
UnsatisfiableJustified ==
    Compose(Lowered(design)).outcome = "unsatisfiable" => \A inst \in Instances : ~Spec(design, inst)

\* "Unsupported" is a representation limit, never a contradictory contract.
UnsupportedSatisfiable ==
    Compose(Lowered(design)).outcome = "unsupported" => \E inst \in Instances : Spec(design, inst)

\* An empty extras domain lowers to a closed object only when it is exact.
EmptyExtrasIsClosed ==
    LET c == Compose(Lowered(design)) IN
    (c.outcome = "admitted" /\ c.extras = {}) =>
        \A inst \in Instances :
            (\E k \in Keys \ DOMAIN c.fields : inst[k] # Absent) => ~Spec(design, inst)
====
