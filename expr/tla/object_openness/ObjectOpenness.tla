---- MODULE ObjectOpenness ----
\* Bounded comparison of Extend composition rules for open and closed objects.
\* Spec semantics: JSON Schema allOf, where each source's additionalProperties
\* applies to every key that source does not itself declare. The derived type
\* declares the whole composed key set, because Loom Extend inherits fields.
EXTENDS FiniteSets, TLC

CONSTANTS Candidate

Keys == {"id", "note", "x"}
Values == {"zero", "neg", "str"}
Absent == "absent"

Int == {"zero", "neg"}
IntMin0 == {"zero"}
Str == {"str"}
Palette == {Int, IntMin0, Str}

Open == [kind |-> "open", schema |-> Values]
Closed == [kind |-> "closed", schema |-> {}]
Policies == {Open, Closed}

Decls(K) == UNION {[D -> Palette] : D \in SUBSET K}
BaseSources == [decl : Decls({"id", "note"}), policy : Policies]
LocalSources == [decl : Decls({"note"}), policy : Policies]
BaseSeqs == {<<b>> : b \in BaseSources} \cup {<<b, c>> : b, c \in BaseSources}
Designs == [bases : BaseSeqs, local : LocalSources]
Instances == [Keys -> Values \cup {Absent}]

Inter(S) == {v \in Values : \A X \in S : v \in X}

BaseSet(d) == {d.bases[i] : i \in DOMAIN d.bases}
ComposedKeys(d) == DOMAIN d.local.decl \cup UNION {DOMAIN b.decl : b \in BaseSet(d)}
DerivedSource(d) ==
    [decl |-> [k \in ComposedKeys(d) |->
                 IF k \in DOMAIN d.local.decl THEN d.local.decl[k] ELSE Values],
     policy |-> d.local.policy]
Sources(d) == BaseSet(d) \cup {DerivedSource(d)}
Declaring(d, k) == {s \in Sources(d) : k \in DOMAIN s.decl}

SourceAccepts(s, inst) ==
    \A k \in Keys : inst[k] # Absent =>
        IF k \in DOMAIN s.decl THEN inst[k] \in s.decl[k] ELSE inst[k] \in s.policy.schema

SpecAccepts(d, inst) == \A s \in Sources(d) : SourceAccepts(s, inst)

\* "flatten" is the behavior before #1062: a field takes only its declarations
\* and one composed policy, closed when any source is closed, covers the rest.
\* "persource" also applies every non-declaring source's policy to the field.
FlattenField(d, k) == Inter({s.decl[k] : s \in Declaring(d, k)})
PerSourceField(d, k) ==
    Inter({s.decl[k] : s \in Declaring(d, k)}
          \cup {s.policy.schema : s \in Sources(d) \ Declaring(d, k)})

Compose(d) ==
    LET fields == [k \in ComposedKeys(d) |->
                     IF Candidate = "flatten" THEN FlattenField(d, k) ELSE PerSourceField(d, k)]
    IN [fields |-> fields,
        extras |-> Inter({s.policy.schema : s \in Sources(d)}),
        admitted |-> \A k \in ComposedKeys(d) : fields[k] # {}]

LoomAccepts(c, inst) ==
    \A k \in Keys : inst[k] # Absent =>
        IF k \in DOMAIN c.fields THEN inst[k] \in c.fields[k] ELSE inst[k] \in c.extras

\* The implemented admission: every closed base declares every composed key
\* (closedBasesDeclare), and declarations of each key are compatible.
ImplAdmits(d) ==
    /\ \A b \in BaseSet(d) : b.policy.kind = "closed" => ComposedKeys(d) \subseteq DOMAIN b.decl
    /\ \A k \in ComposedKeys(d) : FlattenField(d, k) # {}

VARIABLE design
Init == design \in Designs
Next == UNCHANGED design

\* An admitted composition accepts exactly the documents the sources accept.
Sound ==
    LET c == Compose(design) IN
    c.admitted => \A inst \in Instances : LoomAccepts(c, inst) = SpecAccepts(design, inst)

\* A rejected field is one the sources forbid outright.
RejectionJustified ==
    LET c == Compose(design) IN
    \A k \in ComposedKeys(design) :
        c.fields[k] = {} => \A inst \in Instances : inst[k] # Absent => ~SpecAccepts(design, inst)

\* The implemented rule admits exactly the designs the per-source composition admits.
ImplementationAgrees == ImplAdmits(design) = Compose(design).admitted
====
