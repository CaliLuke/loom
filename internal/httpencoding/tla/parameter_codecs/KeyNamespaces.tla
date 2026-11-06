------------------------- MODULE KeyNamespaces -------------------------
\* #968 KeySet contract: literal, prefix (deepObject name[...]) and residual
\* (exploded dynamic map, MapParams) query namespaces on one endpoint; design
\* admission, encoder key admission and decoder key ownership.
EXTENDS Naturals, FiniteSets
CONSTANT Policy
VARIABLES bindings, sent, foreign, phase, admitted, ok, wire, decoded
vars == <<bindings, sent, foreign, phase, admitted, ok, wire, decoded>>

\* Candidate bindings. kind: literal keys, prefix namespace, or residual.
\* "ualt" is one union binding whose two alternatives are residual maps: one
\* dynamic claim shared by both alternatives (matching is #1014's).
Catalog == {"a", "f", "mx", "lit", "res", "res2", "ualt", "mp"}
Spec0 == [a |-> [kind |-> "literal", keys |-> {"a"}, prefix |-> "none"],
          f |-> [kind |-> "literal", keys |-> {"f[x]"}, prefix |-> "none"],
          mx |-> [kind |-> "prefix", keys |-> {}, prefix |-> "m"],
          lit |-> [kind |-> "literal", keys |-> {"m[x]"}, prefix |-> "none"],
          res |-> [kind |-> "residual", keys |-> {}, prefix |-> "none"],
          res2 |-> [kind |-> "residual", keys |-> {}, prefix |-> "none"],
          ualt |-> [kind |-> "residual", keys |-> {}, prefix |-> "none"],
          mp |-> [kind |-> "all", keys |-> {}, prefix |-> "none"]]
Keys == {"a", "f[x]", "m[x]", "m[y]", "z"}
InPrefix(k, p) == (p = "m" /\ k \in {"m[x]", "m[y]"})
\* Entries a binding may be asked to carry (encoder input), before admission.
Candidates(b) == CASE Spec0[b].kind = "literal" -> Spec0[b].keys
                   [] Spec0[b].kind = "prefix" -> {"m[x]", "m[y]"}
                   [] OTHER -> {"a", "m[x]", "z"}

Overlap(b, c) ==
  LET sb == Spec0[b] sc == Spec0[c] IN
  \/ sb.kind = "all" \/ sc.kind = "all"
  \/ sb.keys \cap sc.keys # {}
  \/ (sb.kind = "prefix" /\ \E k \in sc.keys : InPrefix(k, sb.prefix))
  \/ (sc.kind = "prefix" /\ \E k \in sb.keys : InPrefix(k, sc.prefix))
  \/ (sb.kind = "prefix" /\ sc.kind = "prefix" /\ sb.prefix = sc.prefix)
  \/ (sb.kind = "residual" /\ sc.kind = "residual" /\ Policy # "two-residual")
\* Design admission: pairwise disjoint namespaces; at most one residual.
Admit(B) == /\ (Policy = "maps-rejected" => \A b \in B : Spec0[b].kind \in {"literal", "all"})
            \* Rejected candidate: count residual alternatives as separate claims.
            /\ (Policy = "alternatives-conflict" => "ualt" \notin B)
            /\ \A b, c \in B : b # c => (~Overlap(b, c) \/ (Policy = "prefix-literal" /\
              {Spec0[b].kind, Spec0[c].kind} = {"prefix", "literal"}))

\* Static ownership of a wire key: literal, then prefix, then the residual.
Owner(B, k) ==
  LET lit == {b \in B : k \in Spec0[b].keys}
      pre == {b \in B : Spec0[b].kind = "prefix" /\ InPrefix(k, Spec0[b].prefix)}
      res == {b \in B : Spec0[b].kind \in {"residual", "all"}}
  IN IF lit # {} THEN lit
     ELSE IF pre # {} THEN pre
     ELSE res
\* Encoder admission: a dynamic entry is sent only if its own binding owns it.
EncodeOK(B, b, e) == \A k \in e : Policy = "no-encode-check" \/ Owner(B, k) = {b}

Init == /\ bindings \in {B \in SUBSET Catalog : Cardinality(B) \in 1..3}
        /\ sent \in [bindings -> SUBSET Keys]
        /\ \A b \in bindings : sent[b] \subseteq Candidates(b)
        /\ foreign \in SUBSET {"z"}
        /\ phase = "admit" /\ admitted = FALSE /\ ok = FALSE
        /\ wire = {} /\ decoded = [b \in bindings |-> {}]

AdmitStep == /\ phase = "admit"
             /\ admitted' = Admit(bindings)
             /\ phase' = IF admitted' THEN "encode" ELSE "rejected"
             /\ UNCHANGED <<bindings, sent, foreign, ok, wire, decoded>>
EncodeStep == /\ phase = "encode"
              /\ ok' = \A b \in bindings : EncodeOK(bindings, b, sent[b])
              /\ wire' = UNION {sent[b] : b \in bindings} \cup foreign
              /\ phase' = IF ok' THEN "decode" ELSE "done"
              /\ UNCHANGED <<bindings, sent, foreign, admitted, decoded>>
\* greedy: a residual map decodes every key, including its siblings' keys.
DecodeStep == /\ phase = "decode"
              /\ decoded' = [b \in bindings |->
                   {k \in wire : IF Policy = "greedy" /\ Spec0[b].kind = "residual" THEN TRUE
                                 ELSE b \in Owner(bindings, k) /\ Cardinality(Owner(bindings, k)) = 1}]
              /\ phase' = "done"
              /\ UNCHANGED <<bindings, sent, foreign, admitted, ok, wire>>
Next == AdmitStep \/ EncodeStep \/ DecodeStep \/ (phase \in {"done", "rejected"} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* Every binding decodes exactly what it sent; a residual map additionally
\* receives foreign keys that no sibling claims (the source contract).
NoStealing == phase = "done" /\ ok =>
  \A b \in bindings :
    decoded[b] = sent[b] \cup (IF Spec0[b].kind \in {"residual", "all"} THEN foreign ELSE {})
\* Every wire key has at most one owner.
UniqueOwner == phase = "done" /\ ok => \A k \in wire : Cardinality(Owner(bindings, k)) <= 1
\* Admission keeps every disjoint configuration (no blanket map rejection).
DisjointAdmitted == phase # "admit" /\ (\A b, c \in bindings : b # c => ~Overlap(b, c)) => admitted
\* A prefix map owns its whole namespace: every name[k] it may carry is its own.
PrefixComplete == phase # "admit" /\ admitted /\ "mx" \in bindings =>
                    \A k \in {"m[x]", "m[y]"} : Owner(bindings, k) = {"mx"}
=============================================================================
