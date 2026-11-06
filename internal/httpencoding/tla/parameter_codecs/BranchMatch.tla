------------------------- MODULE BranchMatch -------------------------
\* #968 per-branch codec contract consumed by #1014's text-union matcher.
\* Every branch is evaluated twice and independently: Schema (membership of
\* the wire under the branch's codec, exact numbers, no Go widths) and Decode
\* (typed Go conversion and validation into a private candidate). The matcher
\* visits every branch, propagates any fatal callback error, and commits only
\* when the schema count and the decoder count are both one with the same
\* branch. Alternatives of one union share that union's key universe; the
\* endpoint has a literal sibling binding "p"; ownership, not admission,
\* keeps that literal key out of the union's dynamic universe.
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Policy
VARIABLES sibling, sender, sentBranch, sentText, other, fatal, phase, admitted,
          pending, scratch, schemaHits, decodeHits, fatalSeen, result
vars == <<sibling, sender, sentBranch, sentText, other, fatal, phase, admitted,
          pending, scratch, schemaHits, decodeHits, fatalSeen, result>>

\* One query binding "n", form explode=true: an integer branch with an
\* int64-like carrier (holds "1" and "9"), a string branch with pattern
\* ^[a-z]+$, and an exploded integer-valued map branch with an int32-like
\* carrier (holds "1"; the schema also admits "9"). The map is dynamic: it
\* reads every key of the union's universe, including "n".
Branches == <<"int", "string", "map">>
BranchSet == {"int", "string", "map"}
Texts == {"1", "9", "a"}
\* Wire: the union's own "n" key, its map entries (key "x"), and an unrelated
\* literal sibling binding's key "p" when the endpoint declares one.
OwnWire == [n |-> IF sentBranch = "map" THEN "none" ELSE sentText,
            x |-> IF sentBranch = "map" THEN sentText ELSE "none",
            p |-> "none"]
Wire == [OwnWire EXCEPT !.p = other]
\* Keys a branch reads. The map branch reads the whole universe; with the
\* KeySet ownership the universe excludes the sibling's literal key ("p");
\* greedy-ownership is the rejected candidate that lets the map read it.
MapKeys(w) == {k \in {"n", "x", "p"} :
                 /\ w[k] # "none"
                 /\ (k = "p" => Policy = "greedy-ownership")
                 /\ (k = "n" => Policy # "alternatives-as-siblings")}
Schema(b, w) == CASE b = "int" -> w.n \in {"1", "9"}
                  [] b = "string" -> w.n = "a"
                  [] OTHER -> MapKeys(w) # {} /\ \A k \in MapKeys(w) : w[k] \in {"1", "9"}
Decode(b, w) == CASE b = "int" -> Schema(b, w)
                  [] b = "string" -> Schema(b, w)
                  [] OTHER -> Schema(b, w) /\ \A k \in MapKeys(w) : w[k] = "1"
Value(b, w) == CASE b = "map" -> MapKeys(w) [] OTHER -> {w.n}
\* A failing candidate's partial write (shared-scratch candidate only).
Partial(b, w) == IF b = "map" THEN {"entries"} ELSE {}

Init == /\ sibling \in BOOLEAN
        /\ sender \in {"loom", "foreign"}
        /\ sentBranch \in BranchSet
        /\ sentText \in Texts
        /\ other \in IF sibling THEN {"none", "a"} ELSE {"none"}
        /\ fatal \in IF sender = "foreign" THEN {"none", "int", "string"} ELSE {"none"}
        \* A Loom client publishes only after re-matching its own wire.
        /\ sender = "loom" => /\ Decode(sentBranch, OwnWire)
                              /\ {b \in BranchSet : Schema(b, OwnWire)} = {sentBranch}
        /\ phase = "admit" /\ admitted = FALSE /\ pending = Branches
        /\ scratch = {} /\ schemaHits = <<>> /\ decodeHits = <<>> /\ fatalSeen = FALSE
        /\ result = [kind |-> "none", branch |-> "none", value |-> {}]

\* Endpoint admission: a literal sibling beside one dynamic claim is admitted
\* (CheckNamespaces rejects only overlapping literals/prefixes or a second
\* dynamic binding).
Admit == /\ phase = "admit"
         /\ admitted' = TRUE
         /\ phase' = IF admitted' THEN "match" ELSE "rejected"
         /\ UNCHANGED <<sibling, sender, sentBranch, sentText, other, fatal, pending,
                        scratch, schemaHits, decodeHits, fatalSeen, result>>
\* Visit every branch: both callbacks, private candidates only.
Try == /\ phase = "match" /\ pending # <<>>
       /\ LET b == Head(pending)
              f == fatal = b
              s == Schema(b, Wire)
              d == IF Policy = "schema-only" THEN s ELSE Decode(b, Wire)
          IN /\ schemaHits' = IF s /\ ~f THEN Append(schemaHits, b) ELSE schemaHits
             /\ decodeHits' = IF d /\ ~f THEN Append(decodeHits, b) ELSE decodeHits
             /\ fatalSeen' = (fatalSeen \/ (f /\ Policy # "fatal-as-mismatch"))
             /\ scratch' = IF Policy = "shared-scratch" /\ ~d THEN scratch \cup Partial(b, Wire) ELSE scratch
       /\ pending' = IF Policy = "first-match" /\ Decode(Head(pending), Wire) THEN <<>> ELSE Tail(pending)
       /\ UNCHANGED <<sibling, sender, sentBranch, sentText, other, fatal, phase, admitted, result>>
Commit == /\ phase = "match" /\ pending = <<>>
          /\ result' =
               IF fatalSeen THEN [kind |-> "fatal", branch |-> "none", value |-> {}]
               ELSE IF Policy = "first-match" /\ Len(decodeHits) >= 1
                    THEN [kind |-> "value", branch |-> decodeHits[1], value |-> Value(decodeHits[1], Wire)]
               ELSE IF Policy = "decoder-only" /\ Len(decodeHits) = 1
                    THEN [kind |-> "value", branch |-> decodeHits[1], value |-> Value(decodeHits[1], Wire) \cup scratch]
               ELSE IF Len(schemaHits) = 1 /\ Len(decodeHits) = 1 /\
                       (schemaHits[1] = decodeHits[1] \/ Policy = "identity-mismatch")
                    THEN [kind |-> "value", branch |-> decodeHits[1], value |-> Value(decodeHits[1], Wire) \cup scratch]
               ELSE [kind |-> "reject", branch |-> "none", value |-> {}]
          /\ phase' = "done"
          /\ UNCHANGED <<sibling, sender, sentBranch, sentText, other, fatal, admitted,
                         pending, scratch, schemaHits, decodeHits, fatalSeen>>
Next == Admit \/ Try \/ Commit \/ (phase \in {"done", "rejected"} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* Source semantics over the union's shared universe (oneOf exactly-one).
SourceSchema == {b \in BranchSet :
                   CASE b = "int" -> Wire.n \in {"1", "9"}
                     [] b = "string" -> Wire.n = "a"
                     [] OTHER -> LET ks == {k \in {"n", "x"} : Wire[k] # "none"}
                                 IN ks # {} /\ \A k \in ks : Wire[k] \in {"1", "9"}}
SourceDecode == {b \in SourceSchema :
                   CASE b = "int" -> TRUE
                     [] b = "string" -> TRUE
                     [] OTHER -> \A k \in {"n", "x"} : Wire[k] \in {"none", "1"}}

\* A Loom-encoded value without fatal callbacks decodes to its own branch.
LoomRoundTrip == phase = "done" /\ sender = "loom" =>
                   result = [kind |-> "value", branch |-> sentBranch, value |-> Value(sentBranch, Wire)]
\* A committed value is exactly the selected branch's own decode.
CommittedValueOwned == phase = "done" /\ result.kind = "value" => result.value = Value(result.branch, Wire)
\* Commit happens only for one schema member that is also the one decoder
\* member; schema ambiguity is never resolved by carrier overflow or order.
ExactOneBothDomains == phase = "done" /\ result.kind = "value" =>
                         SourceSchema = {result.branch} /\ SourceDecode = {result.branch}
\* A wire with exactly one schema and decoder member, and no fatal error, is
\* committed (no over-rejection, e.g. by hiding alternatives' keys).
UniqueCommitted == phase = "done" /\ fatal = "none" /\ SourceSchema = SourceDecode /\
                   Cardinality(SourceSchema) = 1 =>
                     result.kind = "value"
\* Any fatal callback error is the outcome, whatever other branches report.
FatalPropagated == phase = "done" /\ fatal # "none" => result.kind = "fatal"
=============================================================================
