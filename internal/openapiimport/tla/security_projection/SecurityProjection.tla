-------------------------- MODULE SecurityProjection --------------------------
\* OpenAPI 3.2 Security Requirement keys (#957): the import normalizer composed
\* with the exporter's scheme-global openapi:security:uri rewrite. The oracle
\* resolves each source key independently: component names from the entry
\* document (§4.30, Appendix G.3) and URIs against the containing document's base
\* (RFC 3986 §5.1). The regenerated document is resolved the same way.
\* Authorization meaning (§4.30): every scheme in one requirement object must be
\* satisfied, with all of its listed scopes or roles. Two keys that identify one
\* scheme therefore require the union of their scope sets.
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANT Mode
ASSUME Mode \in {"checked", "authoredCollapse", "keepMixed", "entryBase",
                 "rejectUnequal", "dropAlias", "intersect"}

\* Key spellings:
\*   K        entry component name
\*   absH     https://e.test/h.yaml#/components/securitySchemes/S
\*   relH     h.yaml#/components/securitySchemes/S
\*   absSelfK https://e.test/openapi.json#/components/securitySchemes/K
Spellings == {"K", "absH", "relH", "absSelfK"}
\* Each key lists a scope set; {"a", "b"} arises only from merging aliases.
Scopes == {{}, {"a"}, {"b"}}
\* Requirements live in the entry or in an external Path Item document D whose
\* base is https://e.test/d.yaml. The entry is retrieved from file:/root/; its
\* $self, when present, is https://e.test/openapi.json.
Docs == {"entry", "D"}
\* Targets: entry K, S in the supplied https document h.yaml, S in the file
\* file:/root/h.yaml, or a failed resolution.
VARIABLES entrySelf, reqs, phase, accepted, comps, norm, out
vars == <<entrySelf, reqs, phase, accepted, comps, norm, out>>

HttpsBase(doc) == doc = "D" \/ entrySelf

ResolveURI(sp, httpsBase) ==
    CASE sp = "absH" -> "hS"
      [] sp = "relH" -> IF httpsBase THEN "hS" ELSE "fS"
      [] sp = "absSelfK" -> IF entrySelf THEN "eK" ELSE "fail"
      [] OTHER -> "fail"
\* Oracle: source meaning of a key in its containing document.
SourceTarget(sp, doc) == IF sp = "K" THEN "eK" ELSE ResolveURI(sp, HttpsBase(doc))
\* entryBase is the rejected alternative that resolves URIs against the entry.
ImportTarget(sp, doc) ==
    IF sp = "K" THEN "eK"
    ELSE ResolveURI(sp, IF Mode = "entryBase" THEN entrySelf ELSE HttpsBase(doc))

Occs(r) == {[sp |-> sp, scope |-> r.keys[sp], t |-> ImportTarget(sp, r.doc)] : sp \in DOMAIN r.keys}
AllOccs == UNION {Occs(reqs[i]) : i \in 1..Len(reqs)}
NameOf(t) == CASE t = "eK" -> "K" [] t = "fS" -> "S_f" [] t = "hS" -> "S_h" [] OTHER -> "unresolved"
SpellingsOf(t) == {o.sp : o \in {x \in AllOccs : x.t = t}}
\* A spelling survives only if the regenerated document resolves it identically.
RegenTarget(sp) == ResolveURI(sp, entrySelf)
MetaFor(t) ==
    CASE t = "hS" ->
           IF Mode = "authoredCollapse" THEN CHOOSE sp \in SpellingsOf("hS") : sp # "absH" \/ SpellingsOf("hS") = {"absH"}
           ELSE IF Cardinality(SpellingsOf("hS")) = 1 /\ RegenTarget(CHOOSE sp \in SpellingsOf("hS") : TRUE) = "hS"
                THEN CHOOSE sp \in SpellingsOf("hS") : TRUE
                ELSE "absH"
      [] t = "eK" /\ Mode = "keepMixed" /\ "absSelfK" \in SpellingsOf("eK") -> "absSelfK"
      [] OTHER -> "none"
\* The former V1 rule: aliases of one scheme must list equal scopes.
Consistent(r) == \A o1, o2 \in Occs(r) : o1.t = o2.t => o1.scope = o2.scope
Intersection(S) == {x \in UNION S : \A y \in S : x \in y}
\* Scope sets of the keys that one requirement object maps to target t,
\* merged by union (checked) or by a rejected alternative.
Combine(S) == CASE Mode = "intersect" -> Intersection(S)
                [] Mode = "dropAlias" -> CHOOSE x \in S : TRUE
                [] OTHER -> UNION S
Merged(r, t) == Combine({o.scope : o \in {x \in Occs(r) : x.t = t}})

Init ==
    /\ entrySelf \in BOOLEAN
    /\ reqs \in [1..2 -> [doc : Docs, keys : UNION {[S -> Scopes] : S \in SUBSET Spellings}]]
    /\ phase = "import"
    /\ accepted = FALSE
    /\ comps = {}
    /\ norm = <<>>
    /\ out = <<>>

Import ==
    /\ phase = "import"
    /\ accepted' = /\ \A o \in AllOccs : o.t # "fail"
                   /\ Mode = "rejectUnequal" => \A i \in 1..Len(reqs) : Consistent(reqs[i])
    /\ comps' = {[name |-> "K", t |-> "eK", meta |-> MetaFor("eK")]}
                \cup {[name |-> NameOf(t), t |-> t, meta |-> MetaFor(t)] :
                      t \in {o.t : o \in {x \in AllOccs : x.t \in {"fS", "hS"}}}}
    /\ norm' = [i \in 1..Len(reqs) |-> {[name |-> NameOf(t), scope |-> Merged(reqs[i], t)] :
                                         t \in {o.t : o \in Occs(reqs[i])}}]
    /\ phase' = "export"
    /\ UNCHANGED <<entrySelf, reqs, out>>

\* Exporter: delete every component with URI metadata and rekey every
\* requirement entry that names it (http/codegen/openapi/v3/openapi32.go).
Rekey(name) == LET c == {x \in comps : x.name = name}
               IN IF c # {} /\ (CHOOSE x \in c : TRUE).meta # "none" THEN (CHOOSE x \in c : TRUE).meta ELSE name
Export ==
    /\ phase = "export" /\ accepted
    /\ out' = [i \in 1..Len(norm) |-> {[key |-> Rekey(e.name), scope |-> e.scope] : e \in norm[i]}]
    /\ phase' = "check"
    /\ UNCHANGED <<entrySelf, reqs, accepted, comps, norm>>

Rejected == phase = "export" /\ ~accepted /\ phase' = "rejected"
            /\ UNCHANGED <<entrySelf, reqs, accepted, comps, norm, out>>
Terminated == phase \in {"check", "rejected"} /\ UNCHANGED vars
Next == Import \/ Export \/ Rejected \/ Terminated
Spec == Init /\ [][Next]_vars

\* Regenerated resolution: component names first, then URIs against the
\* regenerated base ($self when the entry declared an absolute one).
Remaining == {x \in comps : x.meta = "none"}
RegenResolve(key) ==
    IF \E x \in Remaining : x.name = key THEN (CHOOSE x \in Remaining : x.name = key).t
    ELSE IF key = "absSelfK" THEN (IF entrySelf /\ \E x \in Remaining : x.name = "K" THEN "eK" ELSE "missing")
    ELSE IF key \in {"absH", "relH"} THEN (IF RegenTarget(key) = "hS" THEN "hS" ELSE "missing")
    ELSE "missing"

\* Meaning of a requirement object: each required scheme with the union of the
\* scope sets of every key that identifies it.
SourceTargets(i) == {SourceTarget(sp, reqs[i].doc) : sp \in DOMAIN reqs[i].keys}
SourceMeaning(i) ==
    {[t |-> t, scope |-> UNION {reqs[i].keys[sp] : sp \in {q \in DOMAIN reqs[i].keys : SourceTarget(q, reqs[i].doc) = t}}] :
       t \in SourceTargets(i)}
RegenMeaning(i) ==
    {[t |-> t, scope |-> UNION {e.scope : e \in {x \in out[i] : RegenResolve(x.key) = t}}] :
       t \in {RegenResolve(e.key) : e \in out[i]}}

\* Requirement order and alternatives are kept; every requirement object
\* requires the same schemes with the same scope sets.
Equivalent == phase = "check" => Len(out) = Len(reqs) /\ \A i \in 1..Len(reqs) : RegenMeaning(i) = SourceMeaning(i)
\* Single-spelling preservation: the authored key survives when it still resolves.
SpellingPreserved ==
    phase = "check" =>
        \A i \in 1..Len(reqs) : \A sp \in DOMAIN reqs[i].keys :
            (SourceTarget(sp, reqs[i].doc) = "hS" /\ SpellingsOf("hS") = {sp} /\ RegenTarget(sp) = "hS")
                => \E e \in out[i] : e.key = sp
\* Only keys that resolve to no scheme justify rejection.
NoFalseRejection ==
    phase = "rejected" => \E i \in 1..Len(reqs), sp \in Spellings :
                              sp \in DOMAIN reqs[i].keys /\ SourceTarget(sp, reqs[i].doc) = "fail"
\* Concrete alias input: one requirement in D lists absH with [a] and relH with
\* [b]; both identify the https scheme S.
ABInput == \E i \in 1..Len(reqs) : reqs[i] = [doc |-> "D", keys |-> [absH |-> {"a"}, relH |-> {"b"}]]
EquivalentOnAB == ABInput => Equivalent
\* Witness (expected to be "violated"): a regenerated run on ABInput exists.
UnionWitness == ~(phase = "check" /\ ABInput)
=============================================================================
