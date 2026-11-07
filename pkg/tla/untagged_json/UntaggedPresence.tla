------------------------ MODULE UntaggedPresence ------------------------
\* Union presence for #1014 in two contexts with different owners.
\*
\* Parameter context (query namespace, #968): branch A is a scalar named "id";
\* branch B is a form-exploded flat object whose optional properties x, y are
\* top-level keys. Undeclared keys (z) belong to siblings and are ignored. The
\* union is present when any exposed key is present; a branch none of whose
\* keys is present was not sent. Rule A (#968): an empty exploded object has
\* no keys, so optional bindings send it as absence and required bindings
\* reject it before sending.
\*
\* Body context (form or multipart root, media owner): presence is body
\* presence, not key presence. A is an open object requiring x; B is a closed
\* empty object. A present body, empty included, is decoded by every branch.
\* For an optional body the media owner decides whether a zero-length body is
\* absence ("absent", e.g. a zero-length form) or an empty value ("value", e.g.
\* an empty multipart envelope); a required body is always present.
EXTENDS Naturals, FiniteSets
CONSTANTS Context, BranchPresence, ClientRuleA, OptionalEmptyBody

R(k, s) == [k |-> k, s |-> s]
Absent == R("absent", {})
Missing == R("missing", {})
Rejected == R("rejected", {})
EncoderError == R("encoder-error", {})
EmptyB == R("B", {})

KA == IF Context = "parameter" THEN {"id"} ELSE {"x"}
KB == IF Context = "parameter" THEN {"x", "y"} ELSE {}
Names == IF Context = "parameter" THEN {"id", "x", "y", "z"} ELSE {"x", "y"}
Intents == IF Context = "parameter"
              THEN {Absent, R("A", {"id"})} \cup {R("B", s) : s \in SUBSET KB}
              ELSE {Absent, R("A", {"x"}), R("A", {"x", "y"}), EmptyB}

\* A wire is a presence bit plus the key set. Parameters have no separate bit:
\* presence is derived from keys. A zero-length optional body may read absent.
Encode(i, required) ==
  IF i = Absent THEN [present |-> FALSE, keys |-> {}]
  ELSE IF Context = "body" /\ i.s = {} /\ ~required /\ OptionalEmptyBody = "absent"
          THEN [present |-> FALSE, keys |-> {}]
  ELSE [present |-> (Context = "body" \/ i.s # {}), keys |-> i.s]

\* Per-branch decoding. The rejected candidates: "empty-is-value" decodes a
\* parameter branch with no keys as an empty object; "param-rule" applies the
\* parameter not-sent rule to root bodies.
AOK(w) == \E a \in KA : a \in w.keys
BOK(w) == IF Context = "parameter"
             THEN IF BranchPresence = "empty-is-value" THEN TRUE ELSE w.keys \cap KB # {}
             ELSE IF BranchPresence = "param-rule" THEN w.keys \cap KB # {} ELSE w.keys = {}
\* "key-count" reproduces the ordinary FormRequest decoder's lost raw-body
\* presence. A required encoded empty object and nonempty separators have no
\* keys but remain present according to the media owner.
Present(w) == IF Context = "parameter" THEN w.keys \cap (KA \cup KB) # {}
              ELSE IF BranchPresence = "key-count" THEN w.keys # {} ELSE w.present

\* The server algorithm, also run by the client on its own encoding.
Decode(w, required) ==
  IF ~Present(w) THEN (IF required THEN Missing ELSE Absent)
  ELSE LET hits == (IF AOK(w) THEN {"A"} ELSE {}) \cup (IF BOK(w) THEN {"B"} ELSE {})
       IN IF Cardinality(hits) # 1 THEN Rejected
          ELSE IF hits = {"A"} THEN R("A", w.keys \cap (IF Context = "parameter" THEN KA ELSE Names))
          ELSE R("B", w.keys \cap KB)

\* The specified observation for each intent.
Expected(i, required) ==
  IF i = EmptyB /\ Context = "parameter" THEN (IF required THEN EncoderError ELSE Absent)
  ELSE IF i = EmptyB /\ ~required /\ OptionalEmptyBody = "absent" THEN Absent
  ELSE i

VARIABLES required, intent, wire, phase, sent, outcome, injected
vars == <<required, intent, wire, phase, sent, outcome, injected>>

Init == /\ required \in BOOLEAN
        /\ intent = Absent /\ wire = [present |-> FALSE, keys |-> {}] /\ phase = "choose"
        /\ sent = FALSE /\ outcome = R("none", {}) /\ injected = FALSE

Choose == /\ phase = "choose"
          /\ \E i \in Intents :
               /\ ~(required /\ i = Absent)
               /\ intent' = i
               /\ wire' = Encode(i, required)
          /\ phase' = "check"
          /\ UNCHANGED <<required, sent, outcome, injected>>

Inject == /\ phase = "choose"
          /\ \E p \in BOOLEAN, s \in SUBSET Names : wire' = [present |-> p, keys |-> s]
          /\ injected' = TRUE /\ phase' = "decode"
          /\ UNCHANGED <<required, intent, sent, outcome>>

\* Client admission: publish when the local decode reproduces the intent, or,
\* under Rule A, when an optional value coincides with absence.
Check == /\ phase = "check"
         /\ LET r == Decode(wire, required) IN
              IF r = intent \/ (ClientRuleA /\ ~required /\ r = Absent)
                 THEN sent' = TRUE /\ phase' = "decode" /\ outcome' = outcome
                 ELSE sent' = FALSE /\ phase' = "done" /\ outcome' = EncoderError
         /\ UNCHANGED <<required, intent, wire, injected>>

Receive == /\ phase = "decode"
           /\ outcome' = Decode(wire, required)
           /\ phase' = "done"
           /\ UNCHANGED <<required, intent, wire, sent, injected>>

Next == Choose \/ Inject \/ Check \/ Receive
Spec == Init /\ [][Next]_vars

\* Sent values are observed exactly as specified.
Agreement == phase = "done" /\ sent => outcome = Expected(intent, required)
\* Every value with a distinguishable spelling, or an optional coincidence
\* with absence, is published.
NoFalseRejection == phase = "done" /\ ~injected /\ Expected(intent, required) # EncoderError => sent
\* Values without a spelling fail before publication.
UnspellableRejected == phase = "done" /\ Expected(intent, required) = EncoderError => ~sent
\* A present empty body is a value, never a missing body.
EmptyBodyIsValue == phase = "done" /\ (sent \/ injected) /\ Context = "body" /\ wire.present /\ wire.keys = {} =>
                      outcome = EmptyB
=============================================================================
