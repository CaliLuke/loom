------------------------- MODULE ResolutionIdentity -------------------------
\* Description resolver identity for #957: lazy loading, alias registration,
\* target keys over internal document keys, optional URI bases, and the fixed
\* point before an occurrence may fail. The operational worklist is compared
\* with a declarative least fixed point.
EXTENDS Naturals, FiniteSets, TLC

CONSTANT Mode
ASSUME Mode \in {"checked", "spellingKeys", "earlyFail", "noBaseCheck",
                 "canonicalDedupe", "aliasShortcut", "inventBase"}

None == "none"
Sources == {"e", "b", "c"}             \* internal document keys: one per loaded source
\* Explicit load policy: Fe/Fb/Fc are file: URIs under the reference root, which
\* exists only when the entry was retrieved from a file. Hb is an https URI owned
\* by a prefix mapping whose directory contains b's file.
LoadURIs == {"Fe", "Fb", "Hb", "Fc"}
Loadable(u) == CASE u = "Fe" -> "e" [] u = "Fb" -> "b" [] u = "Hb" -> "b" [] u = "Fc" -> "c"
CanLoad(u, retrieved) == u \in LoadURIs /\ (u = "Hb" \/ retrieved)
\* "X" is an unsupplied absolute https URI (https://e.test/openapi.json) that a
\* document can claim as $self. "RelSelf" is a relative entry $self.
Absolute == {"X", "Hb"}
\* Fb2 spells Fb with dot segments; normalization (RFC 3986 §6.2.2) removes them.
Normalize(s) == IF s = "Fb2" THEN "Fb" ELSE s
\* "p" and "pe" are two fragment spellings of one decoded pointer (FragmentPointer).
PtrKey(p) == "p"
\* Entry spellings: "Here" is fragment-only; "Rel" is the relative reference
\* "b.yaml#..."; the others are absolute after resolution.
EntryOccs == {[from |-> "e", spell |-> s, ptr |-> p] :
              s \in {"Here", "Rel", "Fe", "Fb", "Fb2", "Hb", "Fc", "X"}, p \in {"p", "pe"}}
OtherOccs(src) == {[from |-> src, spell |-> s, ptr |-> "p"] :
                   s \in {"Here", "Fe", "Fb", "Hb", "Fc", "X"}}

VARIABLES entryRetrieved, selfOf, occs, loaded, alias, active, resolved, failed,
          transplanted, fatal, done
vars == <<entryRetrieved, selfOf, occs, loaded, alias, active, resolved, failed,
          transplanted, fatal, done>>

\* The entry's URI base is optional. An absolute $self is the base; a relative
\* $self resolves against the retrieval URI to Fs; a byte-only entry without an
\* absolute $self has no base. The document key "e" exists in every case.
EntryBase(retrieved) ==
    CASE selfOf["e"] \in Absolute -> selfOf["e"]
      [] retrieved /\ selfOf["e"] = "RelSelf" -> "Fs"
      [] retrieved -> "Fe"
      [] OTHER -> None
EntryIds(retrieved) ==
    (IF retrieved THEN {"Fe"} ELSE {})
    \cup (CASE selfOf["e"] \in Absolute -> {selfOf["e"]}
            [] retrieved /\ selfOf["e"] = "RelSelf" -> {"Fs"}
            [] OTHER -> {})
\* Sibling "b.yaml" of a file base is Fb; of an https://e.test/ base it is Hb.
RelTarget(base) == CASE base = None -> "NOBASE" [] base \in {"Fe", "Fs"} -> "Fb" [] OTHER -> "Hb"
Target(o, retrieved) == IF o.spell = "Rel" THEN RelTarget(EntryBase(retrieved)) ELSE Normalize(o.spell)
\* inventBase is the rejected alternative that supplies a default base (and file
\* confinement) to a byte-only entry.
EffRetrieved == entryRetrieved \/ Mode = "inventBase"

Ids(s, u) == {u} \cup (IF selfOf[s] \in Absolute THEN {selfOf[s]} ELSE {})
AliasDom == {pair[1] : pair \in alias}
AliasSrc(u) == CHOOSE s \in Sources : <<u, s>> \in alias
LoadedURIs == {l.ret : l \in loaded}
LoadedSrcs == {l.src : l \in loaded}
ResolvedOccs == {r.occ : r \in resolved}
Pending == active \ (ResolvedOccs \cup failed)
TargetURI(o) == Target(o, EffRetrieved)
\* Target keys use the internal document key. The rejected noBaseCheck
\* alternative gives each retrieval URI of a selfless file its own document.
KeyDoc(o, s) == IF Mode = "noBaseCheck" /\ s # "e" /\ selfOf[s] = None /\ o.spell # "Here"
                THEN <<s, TargetURI(o)>> ELSE <<s, "">>

Init ==
    /\ entryRetrieved \in BOOLEAN
    /\ selfOf \in {f \in [Sources -> {None, "X", "Hb", "RelSelf"}] :
                     f["b"] # "RelSelf" /\ f["c"] # "RelSelf"}
    /\ occs \in {[s \in Sources |-> CASE s = "e" -> E [] s = "b" -> B [] s = "c" -> C] :
                   E \in {S \in SUBSET EntryOccs : Cardinality(S) <= 2},
                   B \in {{}} \cup {{o} : o \in OtherOccs("b")},
                   C \in {{}} \cup {{o} : o \in OtherOccs("c")}}
    /\ loaded = {[src |-> "e", ret |-> IF EffRetrieved THEN "Fe" ELSE "bytes"]}
    /\ alias = {<<id, "e">> : id \in EntryIds(EffRetrieved)}
    /\ active = occs["e"]
    /\ resolved = {}
    /\ failed = {}
    /\ transplanted = {}
    /\ fatal = FALSE
    /\ done = FALSE

\* Load every referenced URI that explicit policy can open, then register all of
\* the source's identities. Conflicts are checked against loaded sources, never
\* against bytes or canonical spellings.
Load(o) ==
    /\ ~done /\ ~fatal /\ o \in Pending /\ o.spell # "Here"
    /\ CanLoad(TargetURI(o), EffRetrieved)
    /\ TargetURI(o) \notin LoadedURIs
    /\ LET u == TargetURI(o)
           s == Loadable(u)
           ids == Ids(s, u)
           idConflict == \E id \in ids, t \in Sources : <<id, t>> \in alias /\ t # s
           baseConflict == \E l \in loaded : l.src = s /\ l.ret # u /\ selfOf[s] = None
           \* canonicalDedupe keeps the first claimant of an identity.
           added == IF Mode = "canonicalDedupe"
                    THEN {<<id, s>> : id \in {i \in ids : i \notin AliasDom}}
                    ELSE {<<id, s>> : id \in ids}
       IN /\ loaded' = loaded \cup {[src |-> s, ret |-> u]}
          /\ alias' = alias \cup added
          /\ active' = IF s \in LoadedSrcs THEN active ELSE active \cup occs[s]
          /\ fatal' = CASE Mode = "canonicalDedupe" -> baseConflict
                        [] Mode = "noBaseCheck" -> idConflict
                        [] OTHER -> idConflict \/ baseConflict
    /\ UNCHANGED <<entryRetrieved, selfOf, occs, resolved, failed, transplanted, done>>

\* A target key is built only after alias resolution. A loadable URI is resolved
\* only after it is loaded, so an alias cannot hide a competing source. A
\* fragment-only reference needs no base (RFC 3986 §4.4).
Resolve(o) ==
    /\ ~done /\ ~fatal /\ o \in Pending
    /\ o.spell = "Here" \/
         /\ TargetURI(o) \in AliasDom
         /\ (Mode # "aliasShortcut" /\ CanLoad(TargetURI(o), EffRetrieved)) => TargetURI(o) \in LoadedURIs
    /\ LET s == IF o.spell = "Here" THEN o.from ELSE AliasSrc(TargetURI(o))
           key == IF Mode = "spellingKeys" THEN <<<<o.spell, "">>, o.ptr>>
                  ELSE <<KeyDoc(o, s), PtrKey(o.ptr)>>
       IN /\ resolved' = resolved \cup {[occ |-> o, key |-> key, src |-> s]}
          /\ transplanted' = IF s = "e" THEN transplanted
                             ELSE transplanted \cup {[key |-> key, src |-> s, ptr |-> PtrKey(o.ptr)]}
    /\ UNCHANGED <<entryRetrieved, selfOf, occs, loaded, alias, active, failed, fatal, done>>

LoadEnabled == \E o \in Pending : o.spell # "Here" /\ CanLoad(TargetURI(o), EffRetrieved)
                                  /\ TargetURI(o) \notin LoadedURIs

\* OAS 3.2 §4.1.2.1: no reference is unresolvable before every document is
\* parsed. A relative reference from an entry without a base ("NOBASE") is
\* neither an alias nor loadable, so it fails at the fixed point.
Fail(o) ==
    /\ ~done /\ ~fatal /\ o \in Pending /\ o.spell # "Here"
    /\ TargetURI(o) \notin AliasDom
    /\ ~CanLoad(TargetURI(o), EffRetrieved)
    /\ Mode = "earlyFail" \/ ~LoadEnabled
    /\ failed' = failed \cup {o}
    /\ UNCHANGED <<entryRetrieved, selfOf, occs, loaded, alias, active, resolved, transplanted, fatal, done>>

Finish ==
    /\ ~done /\ (fatal \/ Pending = {})
    /\ done' = TRUE
    /\ UNCHANGED <<entryRetrieved, selfOf, occs, loaded, alias, active, resolved, failed, transplanted, fatal>>

Terminated == done /\ UNCHANGED vars

Next == (\E o \in active : Load(o) \/ Resolve(o) \/ Fail(o)) \/ Finish \/ Terminated
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

\* Declarative oracle over the true entry retrieval state.
TargetO(o) == Target(o, entryRetrieved)
OccsOf(S) == UNION {occs[s] : s \in S}
LoadRefs(S) == {o \in OccsOf(S) : o.spell # "Here" /\ CanLoad(TargetO(o), entryRetrieved)}
Step(S) == S \cup {Loadable(TargetO(o)) : o \in LoadRefs(S)}
RECURSIVE Lfp(_)
Lfp(S) == IF Step(S) = S THEN S ELSE Lfp(Step(S))
Reach == Lfp({"e"})
UsedURIs == (IF entryRetrieved THEN {"Fe"} ELSE {}) \cup {TargetO(o) : o \in LoadRefs(Reach)}
Rets(s) == {u \in UsedURIs : Loadable(u) = s}
AllIds(s) == IF s = "e" THEN EntryIds(entryRetrieved) \cup Rets("e")
             ELSE Rets(s) \cup (IF selfOf[s] \in Absolute THEN {selfOf[s]} ELSE {})
FatalExpected ==
    \/ \E s, t \in Reach : s # t /\ AllIds(s) \cap AllIds(t) # {}
    \/ \E s \in Reach : s # "e" /\ selfOf[s] = None /\ Cardinality(Rets(s)) > 1
ExpectedSrc(o) ==
    IF o.spell = "Here" THEN o.from
    ELSE LET C == {s \in Reach : TargetO(o) \in AllIds(s)}
         IN IF C = {} THEN None ELSE CHOOSE s \in C : TRUE

TypeOK == fatal \in BOOLEAN /\ done \in BOOLEAN /\ failed \subseteq active

\* Every completed run agrees with the order-independent fixed point.
FatalMatches == done => (fatal = FatalExpected)
OutcomeMatches ==
    done /\ ~fatal =>
        \A o \in OccsOf(Reach) :
            IF ExpectedSrc(o) = None THEN o \in failed
            ELSE \E r \in resolved : /\ r.occ = o /\ r.src = ExpectedSrc(o)
                                     /\ r.key = <<<<ExpectedSrc(o), "">>, PtrKey(o.ptr)>>
\* One transplant key per (source, decoded pointer), and no key for two targets.
OneKeyPerTarget ==
    \A t1, t2 \in transplanted : (t1.src = t2.src /\ t1.ptr = t2.ptr) <=> (t1.key = t2.key)
TransplantComplete ==
    done /\ ~fatal => \A r \in resolved : r.src # "e" => \E t \in transplanted : t.key = r.key
\* A byte-only entry without an absolute $self registers no URI identity.
NoInventedIdentity ==
    (~entryRetrieved /\ selfOf["e"] \notin Absolute) => \A pr \in alias : pr[2] # "e"
Termination == <>done
=============================================================================
