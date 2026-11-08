-------------------------- MODULE MixedResponse --------------------------
\* Bounded role/capture refinement of SSEOpen and SSETerminal's shared owner.
\* It does not model Go compilation, arbitrary validation callbacks or codecs.
EXTENDS Naturals, Sequences
CONSTANTS Policy, MaxSteps
VARIABLES representation, required, metadata, snapshot, phase, wire, body,
          calls, sends, replacement, setResult, reason, steps
vars == <<representation, required, metadata, snapshot, phase, wire, body,
          calls, sends, replacement, setResult, reason, steps>>
None == [token |-> "none", status |-> 0, header |-> FALSE]
Implicit == [token |-> "implicit", status |-> 201, header |-> TRUE]
Values == {[token |-> "one", status |-> 201, header |-> TRUE],
           [token |-> "two", status |-> 202, header |-> TRUE],
           [token |-> "invalid", status |-> 303, header |-> TRUE],
           [token |-> "missing-field", status |-> 201, header |-> FALSE]}
\* Abstract output of the existing captured domain AND native/mapped validators.
\* There is no new selector equation here.
\* required means a metadata value is required, not a required-header flag.
\* Status values are representative owned tokens, not a literal DSL encoding.
\* header abstracts the remaining native/mapped validity conjunction.
Valid(m) == m.status \in {201, 202} /\ m.header
Selected == IF metadata = None /\ ~required THEN Implicit ELSE metadata
Init == /\ representation \in {"unary", "sse"} /\ required \in BOOLEAN
        /\ metadata = None /\ snapshot = None /\ phase = "before"
        /\ wire = <<>> /\ body = "none" /\ calls = 0 /\ sends = 0
        /\ replacement = FALSE /\ setResult = "none" /\ reason = "none"
        /\ steps = 0
Invoke == /\ phase = "before" /\ steps < MaxSteps
          /\ phase' = "live" /\ calls' = 1 /\ steps' = steps + 1
          /\ UNCHANGED <<representation, required, metadata, snapshot, wire,
                         body, sends, replacement, setResult, reason>>
SetResponse == /\ (phase \in {"live", "committed"} \/ (Policy = "held-set" /\ phase = "held"))
               /\ steps < MaxSteps
               /\ \E m \in Values :
                    LET admitted == phase = "live" \/ Policy = "late-set" \/ (Policy = "held-set" /\ phase = "held")
                        accepted == admitted /\ Valid(m)
                    IN /\ metadata' = IF accepted THEN m ELSE metadata
                       /\ replacement' = (replacement \/ (accepted /\ metadata # None /\ m # metadata))
                       /\ setResult' = IF ~admitted THEN "committed"
                                       ELSE IF Valid(m) THEN "ok" ELSE "invalid"
               /\ steps' = steps + 1
               /\ UNCHANGED <<representation, required, snapshot, phase, wire,
                              body, calls, sends, reason>>
\* All discard stream operations (Send, Close, Open, comments, completion)
\* are represented by this nonpublishing action. Actual SSE open uses Begin.
DiscardSend == /\ phase = "live" /\ representation = "unary" /\ steps < MaxSteps
               /\ sends' = sends + 1 /\ steps' = steps + 1
               /\ IF Policy = "discard-commit" /\ Valid(Selected)
                  THEN /\ wire' = <<Selected>> /\ snapshot' = Selected
                       /\ body' = "result" /\ phase' = "committed" /\ reason' = "discard"
                  ELSE UNCHANGED <<wire, snapshot, body, phase, reason>>
               /\ UNCHANGED <<representation, required, metadata, calls, replacement, setResult>>
\* Shared response-metadata admission owner holds its lock across capture/seal
\* and this abstract publication interval for BOTH unary and SSE. No
\* setter can run in phase held. Failed preparation publishes nothing.
Begin == /\ phase = "live" /\ steps < MaxSteps
         /\ snapshot' = Selected
         /\ phase' = IF Valid(Selected) \/ Policy = "missing-check" THEN "held" ELSE "failed"
         /\ body' = IF representation = "unary"
                    THEN IF Policy = "current-split" THEN "handshake" ELSE "result"
                    ELSE "event"
         /\ reason' = "success" /\ steps' = steps + 1
         /\ UNCHANGED <<representation, required, metadata, wire, calls, sends,
                        replacement, setResult>>
Publish == /\ phase = "held" /\ steps < MaxSteps
           /\ wire' = <<snapshot>> /\ phase' = "committed" /\ steps' = steps + 1
           /\ UNCHANGED <<representation, required, metadata, snapshot, body,
                          calls, sends, replacement, setResult, reason>>
Next == Invoke \/ SetResponse \/ DiscardSend \/ Begin \/ Publish
Spec == Init /\ [][Next]_vars
TypeOK == /\ representation \in {"unary", "sse"} /\ required \in BOOLEAN
          /\ metadata \in Values \cup {None} /\ snapshot \in Values \cup {None, Implicit}
          /\ phase \in {"before", "live", "held", "failed", "committed"}
          /\ body \in {"none", "handshake", "result", "event"}
          /\ calls \in 0..1 /\ sends \in 0..MaxSteps /\ steps \in 0..MaxSteps
          /\ wire \in {<<>>} \cup {<<m>> : m \in Values \cup {None, Implicit}}
          /\ reason \in {"none", "discard", "success"}
          /\ setResult \in {"none", "committed", "ok", "invalid"}
          /\ replacement \in BOOLEAN
\* Structural sanity: the model admits only one invocation action.
SingleInvocation == phase # "before" => calls = 1
SinglePublication == Len(wire) <= 1
PreparedValid == wire # <<>> => Valid(wire[1])
BodySeparation == phase \in {"held", "committed"} =>
                   body = (IF representation = "unary" THEN "result" ELSE "event")
\* Ghost provenance check, not an observation of an actual network write.
PhysicalTrigger == wire # <<>> => reason = "success"
\* Structural sanity: Publish copies the held snapshot by construction.
SnapshotAgreement == wire # <<>> => wire[1] = snapshot
MetadataFrozen == wire # <<>> => metadata = (IF snapshot = Implicit THEN None ELSE snapshot)
NoUnarySuccess == ~(representation = "unary" /\ wire # <<>>)
NoSSESuccess == ~(representation = "sse" /\ wire # <<>>)
NoUnaryTypedAfterDiscard == ~(representation = "unary" /\ required /\ sends > 0
                              /\ wire # <<>> /\ wire[1].token \in {"one", "two"})
NoReplacement == ~replacement
NoInvalidSet == setResult # "invalid"
NoMissingFailure == ~(required /\ metadata = None /\ phase = "failed")
=============================================================================
