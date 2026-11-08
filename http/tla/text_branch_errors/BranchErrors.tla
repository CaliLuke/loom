------------------------- MODULE BranchErrors -------------------------
\* Error provenance through one untagged text-union match: the candidate's
\* assignment closure (a generated adapter or a hand-written TextBranch), the
\* public runtime TextBranch.Decode, and exact-one selection. Selection follows
\* BranchMatch: every candidate's schema and decoder predicates run
\* independently, and the first callback error fails the match with its cause.
\* NestedFailure is the precedent for separating an ordinary mismatch from an
\* unexpected failure.
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Policy

Candidates == <<"A", "B">>
CandidateSet == {"A", "B"}

\* What an assignment closure meets for one candidate. Accepted untagged
\* admission rejects Go replacements anywhere in a branch, so a generated
\* closure holds only conversion and validation of supplied text: an ordinary
\* "mismatch". A hand-written branch reports its own mismatch as (false, nil)
\* and may return any error: an "unknown" evaluator, configuration or codec
\* failure, or a server "fault" such as a faulty authored default.
Kinds(p) == IF p = "generated" THEN {"ok", "mismatch"}
            ELSE {"ok", "mismatch", "unknown", "fault"}

\* checked: generated adapter returns (false, nil); runtime transparent.
\* r3: generated adapter returns its raw error; runtime keeps only faults.
\* raw-adapter: generated adapter returns its raw error; runtime transparent.
\* runtime-swallow: runtime turns every Assign error into a mismatch.
\* schema-gated, drop-unselected: rejected limits on unselected-candidate errors.
Adapter == IF Policy \in {"r3", "raw-adapter"} THEN "raw" ELSE "mismatch"
Runtime == CASE Policy = "r3" -> "keep-faults"
             [] Policy = "runtime-swallow" -> "swallow"
             [] OTHER -> "transparent"

IsErrOut(o) == o \in {"err:fault", "err:mismatch", "err:unknown"}
Cause(o) == CASE o = "err:fault" -> "fault"
              [] o = "err:unknown" -> "unknown"
              [] OTHER -> "mismatch"
\* Closure result: "match", "nomatch" or "err:<cause>".
AssignOut(p, k) ==
    CASE k = "ok" -> "match"
      [] p = "handwritten" -> CASE k = "unknown" -> "err:unknown"
                                [] k = "fault" -> "err:fault"
                                [] OTHER -> "nomatch"
      [] Adapter = "raw" -> "err:mismatch"
      [] OTHER -> "nomatch"
\* TextBranch.Decode returns Assign's result unless a policy suppresses it.
DecodeOut(o) == CASE Runtime = "keep-faults" /\ o \in {"err:mismatch", "err:unknown"} -> "nomatch"
                  [] Runtime = "swallow" /\ IsErrOut(o) -> "nomatch"
                  [] OTHER -> o

VARIABLES producer, schema, kind, pending, schemaHits, decodeHits, phase, result
vars == <<producer, schema, kind, pending, schemaHits, decodeHits, phase, result>>

Init == /\ producer \in [CandidateSet -> {"generated", "handwritten"}]
        /\ schema \in [CandidateSet -> BOOLEAN]
        /\ kind \in [CandidateSet -> {"ok", "mismatch", "unknown", "fault"}]
        /\ \A c \in CandidateSet : kind[c] \in Kinds(producer[c])
        /\ pending = Candidates
        /\ schemaHits = {}
        /\ decodeHits = {}
        /\ phase = "match"
        /\ result = [kind |-> "none", branch |-> "none", cause |-> "none"]

\* One candidate: schema predicate, then decoder predicate (Select order).
Try == /\ phase = "match"
       /\ pending # <<>>
       /\ LET c == Head(pending)
              run == ~(Policy = "schema-gated" /\ ~schema[c])
              o == IF run THEN DecodeOut(AssignOut(producer[c], kind[c])) ELSE "skip"
              dropped == Policy = "drop-unselected" /\ IsErrOut(o) /\ ~schema[c]
          IN IF IsErrOut(o) /\ ~dropped
             THEN /\ result' = [kind |-> "fatal", branch |-> "none", cause |-> Cause(o)]
                  /\ phase' = "done"
                  /\ UNCHANGED <<pending, schemaHits, decodeHits>>
             ELSE /\ schemaHits' = IF schema[c] THEN schemaHits \cup {c} ELSE schemaHits
                  /\ decodeHits' = IF o = "match" THEN decodeHits \cup {c} ELSE decodeHits
                  /\ pending' = Tail(pending)
                  /\ UNCHANGED <<phase, result>>
       /\ UNCHANGED <<producer, schema, kind>>

Commit == /\ phase = "match"
          /\ pending = <<>>
          /\ result' = IF Cardinality(schemaHits) = 1 /\ schemaHits = decodeHits
                       THEN [kind |-> "value", branch |-> CHOOSE c \in schemaHits : TRUE,
                             cause |-> "none"]
                       ELSE [kind |-> "reject", branch |-> "none", cause |-> "none"]
          /\ phase' = "done"
          /\ UNCHANGED <<producer, schema, kind, pending, schemaHits, decodeHits>>

Next == Try \/ Commit \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* Contract oracle, independent of Policy: an error from any candidate's
\* decoder fails the match with the first failing candidate's cause, in
\* candidate order. Ordinary mismatches are never errors.
Fails(c) == kind[c] \in {"fault", "unknown"}
AnyFail == \E c \in CandidateSet : Fails(c)
FirstFail == Candidates[CHOOSE i \in 1..Len(Candidates) :
                          /\ Fails(Candidates[i])
                          /\ \A j \in 1..(i - 1) : ~Fails(Candidates[j])]
SchemaSet == {c \in CandidateSet : schema[c]}
DecodeSet == {c \in CandidateSet : kind[c] = "ok"}

\* A hand-written assignment error reaches the caller with its cause.
UnknownErrorRetained == phase = "done" /\ AnyFail /\ kind[FirstFail] = "unknown" =>
                          result.kind = "fatal" /\ result.cause = "unknown"
\* A server fault from any evaluated candidate fails the match, even when
\* another candidate is the unique schema and decoder match.
FaultFails == phase = "done" /\ AnyFail /\ kind[FirstFail] = "fault" =>
                result.kind = "fatal" /\ result.cause = "fault"
\* Ordinary conversion or validation mismatches never block a unique match.
MismatchIsNotFatal == phase = "done" /\ ~AnyFail /\ Cardinality(SchemaSet) = 1 /\
                      SchemaSet = DecodeSet =>
                        result.kind = "value" /\ {result.branch} = SchemaSet
\* A fatal outcome only reports a contracted failure, with the right cause.
FatalOnlyWhenContracted == phase = "done" /\ result.kind = "fatal" =>
                             AnyFail /\ result.cause = kind[FirstFail]
\* BranchMatch's exact-one rule over both independently evaluated domains.
CommitSound == phase = "done" /\ result.kind = "value" =>
                 SchemaSet = {result.branch} /\ DecodeSet = {result.branch}

\* Reachability witness, checked as an expected violation: a fault from a
\* hand-written candidate that is not the unique schema match fails a request
\* that another candidate would have matched.
NoUnselectedFaultFailure ==
    ~(phase = "done" /\ result.cause = "fault" /\
      \E c \in CandidateSet : SchemaSet = {c} /\ kind[c] = "ok")
=============================================================================
