---- MODULE ImportAllOf ----
\* Translation of raw allOf: [$ref B, S] into DSL Extend(B) with local S.
\* Raw S declares only its own properties. A DSL Extend level inherits B's
\* declarations, so the importer must reconcile S's policy over B-only keys.
EXTENDS FiniteSets, TLC

CONSTANTS Mode

Keys == {"id", "note", "x"}
Values == {"zero", "neg", "str", "null"}
Absent == "absent"
Int == {"zero", "neg"}
Str == {"str"}
Palette == {Int, Str}

Open == [kind |-> "open", schema |-> Values]
Closed == [kind |-> "closed", schema |-> {}]
Policies == {Open, Closed, [kind |-> "typed", schema |-> Int], [kind |-> "typed", schema |-> Str]}
Decls(K) == UNION {[D -> Palette] : D \in SUBSET K}
Raw == [decl : Decls({"id", "note"}), policy : Policies, req : SUBSET {"id", "note"}]
Instances == [Keys -> Values \cup {Absent}]

Accepts(s, inst) ==
    /\ \A k \in s.req : inst[k] # Absent
    /\ \A k \in Keys : inst[k] # Absent =>
         IF k \in DOMAIN s.decl THEN inst[k] \in s.decl[k] ELSE inst[k] \in s.policy.schema
RawSpec(b, s, inst) == Accepts(b, inst) /\ Accepts(s, inst)

\* "unchecked": copy S as the local level. "checked": the current closed-sibling
\* admission plus redeclaring each B-only key under a typed S policy.
Translate(b, s) ==
    LET bOnly == DOMAIN b.decl \ DOMAIN s.decl
        localDecl == IF Mode = "checked" /\ s.policy.kind = "typed"
                     THEN [k \in DOMAIN s.decl \cup bOnly |->
                             IF k \in DOMAIN s.decl THEN s.decl[k] ELSE s.policy.schema]
                     ELSE s.decl
        admitted == Mode = "unchecked" \/ s.policy.kind # "closed" \/ bOnly = {}
    IN [local |-> [decl |-> localDecl, policy |-> s.policy, req |-> s.req], admitted |-> admitted]

\* DSL meaning of Extend(B) + local: the level declares B's keys as well.
DSLSpec(b, l, inst) ==
    LET level == [decl |-> [k \in DOMAIN b.decl \cup DOMAIN l.decl |->
                              IF k \in DOMAIN l.decl THEN l.decl[k] ELSE Values],
                  policy |-> l.policy, req |-> l.req]
    IN Accepts(b, inst) /\ Accepts(level, inst)

VARIABLE pair
Init == pair \in [b : Raw, s : Raw]
Next == UNCHANGED pair

TranslationSound ==
    LET t == Translate(pair.b, pair.s) IN
    t.admitted => \A inst \in Instances : DSLSpec(pair.b, t.local, inst) = RawSpec(pair.b, pair.s, inst)
====
