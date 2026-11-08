---------------------- MODULE DiscriminatorImport ----------------------
\* Representation choice for an imported discriminated oneOf (#1005).
\* Branch 1 varies over every modeled fact; branch 2 over four controls.
\* The classifier inspects each branch, then the document, and selects the
\* tagged envelope, the full-object untagged union with a hint, or a block.
\* Invariants compare the source oneOf domain with the representation's
\* regenerated contract over every modeled JSON value.
EXTENDS Naturals, FiniteSets
CONSTANT Mode

K == <<"k1", "k2">>                \* each branch's effective mapping key
Reps == {"Tagged", "FullObject"}
Blocks == {"optional-needs-defaultMapping", "versioned-field", "optional-before-3.1", "optional-before-3.2"}

Branch1 == [shape : {"envelope", "flat"}, tagEnum : {"key", "other", "free"},
            closed : BOOLEAN, tagExtra : BOOLEAN, requiresP : BOOLEAN,
            sharedRef : BOOLEAN]
Clean(shape) == [shape |-> shape, tagEnum |-> "key", closed |-> FALSE,
                 tagExtra |-> FALSE, requiresP |-> TRUE, sharedRef |-> FALSE]
Branch2 == {Clean("envelope"), Clean("flat"),
            [Clean("envelope") EXCEPT !.requiresP = FALSE],
            [Clean("envelope") EXCEPT !.tagEnum = "other"]}

\* A JSON object value: discriminator value (or absent), an undeclared member,
\* an over-long discriminator, payload validity and payload nesting under the
\* envelope value member (TRUE) versus flat members (FALSE).
Values == [tag : {"k1", "k2", "o", "none"}, extra : BOOLEAN, longTag : BOOLEAN,
           payload : BOOLEAN, nested : BOOLEAN]

VARIABLES b, version, hasDefault, occurrence, unionClosed,
          phase, i, allExact, outcome
vars == <<b, version, hasDefault, occurrence, unionClosed, phase, i, allExact, outcome>>

\* Source semantics: JSON Schema oneOf, branch facts kept as authored.
TagOK(f, v, key) == IF v.tag = "none" THEN ~f.requiresP
                    ELSE CASE f.tagEnum = "key" -> v.tag = key
                           [] f.tagEnum = "other" -> v.tag = "o"
                           [] OTHER -> TRUE
BranchAcc(f, v, key) == /\ TagOK(f, v, key)
                        /\ f.closed => ~v.extra
                        /\ (f.tagExtra /\ v.tag # "none") => ~v.longTag
                        /\ v.payload
                        /\ v.nested = (f.shape = "envelope")
Matches(v) == {j \in 1..2 : BranchAcc(b[j], v, K[j])}
SourceAcc(v) == Cardinality(Matches(v)) = 1 /\ (unionClosed => ~v.extra)

\* Exact envelope: every fact has a tagged-envelope slot (design table).
\* Each lenient mode relaxes one fact, reproducing a review counterexample.
Exact(f) == /\ f.shape = "envelope"
            /\ f.tagEnum = "key" \/ (Mode = "lenient-enum" /\ f.tagEnum = "other")
            /\ ~f.closed \/ Mode = "lenient-closed"
            /\ ~f.tagExtra \/ Mode = "lenient-tag"
            /\ f.requiresP /\ ~f.sharedRef
\* Wire tag Loom assigns to a tagged branch: the envelope's enum value.
WireTag(j) == IF b[j].tagEnum = "other" THEN "o" ELSE K[j]
Optional == \E j \in 1..2 : ~b[j].requiresP

\* Regenerated tagged contract: open envelopes with required tag/value,
\* union-level unevaluatedProperties retained by occurrence metadata. In a
\* form body Loom's tagged codec flattens object branches.
TaggedAcc(v) == /\ \E j \in 1..2 : v.tag = WireTag(j)
                /\ v.payload
                /\ v.nested = (occurrence = "json")
                /\ unionClosed => ~v.extra
\* Full-object union: branches kept as authored, exact-one runtime matching.
\* The rejected hint-routing candidate validates only the mapped branch.
FullAcc(v) == IF Mode = "hint-routing"
                 THEN /\ \E j \in 1..2 : v.tag = K[j] /\ BranchAcc(b[j], v, K[j])
                      /\ unionClosed => ~v.extra
                 ELSE SourceAcc(v)
RepAcc(v) == IF outcome = "Tagged" THEN TaggedAcc(v) ELSE FullAcc(v)

Init == /\ b \in {<<x, y>> : x \in Branch1, y \in Branch2}
        /\ version \in {"3.0", "3.1", "3.2"}   \* importer source versions
        /\ hasDefault \in BOOLEAN
        /\ occurrence \in {"json", "form"}
        /\ unionClosed \in BOOLEAN
        /\ phase = "branches" /\ i = 1 /\ allExact = TRUE /\ outcome = "none"

InspectBranch == /\ phase = "branches" /\ i <= 2
                 /\ allExact' = (allExact /\ Exact(b[i]))
                 /\ i' = i + 1
                 /\ UNCHANGED <<b, version, hasDefault, occurrence, unionClosed, phase, outcome>>

Decide == /\ phase = "branches" /\ i > 2
          /\ outcome' =
               CASE version = "3.2" /\ Optional /\ ~hasDefault -> "optional-needs-defaultMapping"
                 [] version \in {"3.0", "3.1"} /\ hasDefault -> "versioned-field"
                 [] Mode # "same-30-31" /\ version = "3.0" /\ Optional -> "optional-before-3.1"
                 [] Mode = "prior" /\ version = "3.1" /\ Optional -> "optional-before-3.2"
                 [] allExact /\ (occurrence = "json" \/ Mode = "tagged-everywhere") -> "Tagged"
                 [] OTHER -> "FullObject"
          /\ phase' = "done"
          /\ UNCHANGED <<b, version, hasDefault, occurrence, unionClosed, i, allExact>>

Next == InspectBranch \/ Decide
Spec == Init /\ [][Next]_vars

Classified == phase = "done" => outcome \in Reps \cup Blocks
DomainPreserved == phase = "done" /\ outcome \in Reps =>
                     \A v \in Values : SourceAcc(v) = RepAcc(v)
TagAgreement == phase = "done" /\ outcome = "Tagged" => \A j \in 1..2 : WireTag(j) = K[j]
NeverTaggedOptional == phase = "done" /\ outcome = "Tagged" => ~Optional
VersionRules == phase = "done" /\ outcome \in Reps =>
                  /\ version \in {"3.0", "3.1"} => ~hasDefault
                  /\ version = "3.0" => ~Optional
                  /\ (version = "3.2" /\ Optional) => hasDefault
\* OAS 3.1.1 makes a required discriminating property a SHOULD; it is admitted.
\* OAS 3.0.3 (Schema Object, polymorphism) makes it a MUST.
Admits31Optional == phase = "done" /\ version = "3.1" /\ ~hasDefault => outcome \in Reps
=============================================================================
