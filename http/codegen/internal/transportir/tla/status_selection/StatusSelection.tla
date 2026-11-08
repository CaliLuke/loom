------------------------- MODULE StatusSelection -------------------------
\* #976 final-phase response selection: exact, class and fallback selectors,
\* success/error key ownership, server status ownership and client decoding.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
VARIABLES keys, owner, body, phase, kind, branch, sent, bodyOK, wire, outcome
vars == <<keys, owner, body, phase, kind, branch, sent, bodyOK, wire, outcome>>

Codes == {200, 201, 204, 404, 409, 500}
Exact(c) == [kind |-> "exact", code |-> c, class |-> c \div 100]
ClassKey(n) == [kind |-> "class", code |-> 0, class |-> n]
Fallback == [kind |-> "fallback", code |-> 0, class |-> 0]
NoKey == [kind |-> "none", code |-> 0, class |-> 0]
Unmapped == [kind |-> "unmapped", code |-> 0, class |-> 0]
Universe == {Exact(200), ClassKey(2), Exact(404), ClassKey(4), Fallback}
Class(c) == c \div 100
Final(c) == c >= 200 /\ c <= 599
Bodiless(c) == c \in {204, 304}
Specificity(k) == CASE k.kind = "exact" -> 2 [] k.kind = "class" -> 1 [] OTHER -> 0
Covers(k, c) == CASE k.kind = "exact" -> k.code = c
                  [] k.kind = "class" -> Class(c) = k.class
                  [] OTHER -> Final(c)

\* Design algorithm: membership tests in fixed precedence, independent of order.
Select(ks, c) == IF ~Final(c) THEN NoKey
                 ELSE IF Exact(c) \in ks THEN Exact(c)
                 ELSE IF ClassKey(Class(c)) \in ks THEN ClassKey(Class(c))
                 ELSE IF Fallback \in ks THEN Fallback ELSE NoKey
\* Independent oracle: the covering key of greatest specificity.
MostSpecific(ks, c) ==
  LET cov == {k \in ks : Covers(k, c)}
  IN IF cov = {} THEN NoKey ELSE CHOOSE k \in cov : \A j \in cov : Specificity(j) <= Specificity(k)
\* Rejected candidate (a): declaration order with the least specific key first.
RECURSIVE FirstCover(_, _)
FirstCover(s, c) == IF s = <<>> THEN NoKey
                    ELSE IF Covers(Head(s), c) THEN Head(s) ELSE FirstCover(Tail(s), c)
LeastSpecificFirst(ks) ==
  LET f == IF Fallback \in ks THEN <<Fallback>> ELSE <<>>
      cl == [i \in 1..Cardinality({k \in ks : k.kind = "class"})
              |-> CHOOSE k \in {x \in ks : x.kind = "class"} :
                  Cardinality({y \in ks : y.kind = "class" /\ y.class < k.class}) = i - 1]
      ex == [i \in 1..Cardinality({k \in ks : k.kind = "exact"})
              |-> CHOOSE k \in {x \in ks : x.kind = "exact"} :
                  Cardinality({y \in ks : y.kind = "exact" /\ y.code < k.code}) = i - 1]
  IN f \o cl \o ex
ClientSelect(c) == IF Policy = "decl-order" THEN FirstCover(LeastSpecificFirst(keys), c)
                   ELSE Select(keys, c)

\* Server-side runtime domain computed from the design formula, not from Select.
Domain(k) ==
  LET base == CASE k.kind = "exact" -> {k.code}
              [] k.kind = "class" ->
                   {c \in Codes : Class(c) = k.class /\
                                  (Policy = "server-loose" \/ Exact(c) \notin keys)}
              [] OTHER -> {c \in Codes : Exact(c) \notin keys /\ ClassKey(Class(c)) \notin keys}
  IN IF body[k] THEN {c \in base : ~Bodiless(c)} ELSE base

SuccessKeys == {k \in keys : owner[k] \in {"success", "both"}}
ErrorKeys == {k \in keys : owner[k] \in {"error", "both"}}
\* An operation may declare only error keys (no invented success key).
Admitted == Policy # "mixed" => \A k \in keys : owner[k] # "both"

Init == /\ keys \in {s \in SUBSET Universe : Cardinality(s) \in 1..3}
        /\ owner \in [keys -> {"success", "error", "both"}]
        /\ body \in [keys -> BOOLEAN]
        /\ phase = "validate"
        /\ kind = "none" /\ branch = NoKey /\ sent = 0 /\ bodyOK = TRUE
        /\ wire = [status |-> 0, branch |-> NoKey, kind |-> "none", origin |-> "none"]
        /\ outcome = [result |-> "none", key |-> NoKey, status |-> 0, side |-> "none"]

Validate == /\ phase = "validate"
            /\ phase' = IF Admitted THEN "serve" ELSE "rejected"
            /\ UNCHANGED <<keys, owner, body, kind, branch, sent, bodyOK, wire, outcome>>

\* The service chooses a designed branch and a status value; the encoder owns
\* the selection check. Publication records its origin: the authored branch,
\* or the framework's unmapped-error response (status 500) after a violation.
\* The publish-violation candidate commits the authored status unchecked.
Serve == /\ phase = "serve"
         /\ \/ \E k \in {"success", "error"} :
              \E b \in (IF k = "success" THEN SuccessKeys ELSE ErrorKeys) :
                \E v \in Codes, ok \in BOOLEAN :
                  /\ kind' = k /\ branch' = b /\ sent' = v /\ bodyOK' = ok
                  /\ wire' = IF v \in Domain(b) \/ Policy = "publish-violation"
                             THEN [status |-> v, branch |-> b, kind |-> k, origin |-> "authored"]
                             ELSE [status |-> 500, branch |-> Unmapped, kind |-> "unmapped", origin |-> "framework"]
            \* A successful return on an error-only operation has no designed
            \* response: it is a framework violation, never an invented 200.
            \/ /\ SuccessKeys = {}
               /\ \E ok \in BOOLEAN :
                    /\ kind' = "success" /\ branch' = Unmapped /\ sent' = 0 /\ bodyOK' = ok
                    /\ wire' = [status |-> 500, branch |-> Unmapped, kind |-> "unmapped", origin |-> "framework"]
         /\ phase' = "decode"
         /\ UNCHANGED <<keys, owner, body, outcome>>

\* The client selects a key once; every later failure is terminal. bodyOK is
\* whether the received body decodes and validates under the selected branch;
\* an unmapped problem body may or may not satisfy a 5XX/fallback schema.
\* The accept-invalid candidate keeps the selected key but ignores body failure.
DecodeKey(k, c) ==
  IF body[k] /\ Bodiless(c) THEN "invalid"
  ELSE IF bodyOK \/ Policy = "accept-invalid" THEN "decoded" ELSE "invalid"
SideOf(k) == IF owner[k] = "both" THEN "success" ELSE owner[k]
Decode == /\ phase = "decode"
          /\ LET c == wire.status
                 k == ClientSelect(c)
                 fallback == Select(keys \ {k}, c)
             IN outcome' =
                  IF k = NoKey THEN [result |-> "invalid", key |-> NoKey, status |-> c, side |-> "none"]
                  ELSE IF DecodeKey(k, c) = "decoded"
                       THEN [result |-> "value", key |-> k, status |-> c, side |-> SideOf(k)]
                  ELSE IF Policy = "fallthrough" /\ fallback # NoKey /\ ~(body[fallback] /\ Bodiless(c))
                       THEN [result |-> "value", key |-> fallback, status |-> c, side |-> SideOf(fallback)]
                  ELSE [result |-> "invalid", key |-> k, status |-> c, side |-> "none"]
          /\ phase' = "done"
          /\ UNCHANGED <<keys, owner, body, kind, branch, sent, bodyOK, wire>>

Next == Validate \/ Serve \/ Decode \/ (phase \in {"done", "rejected"} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* P1: the client applies exact > class > fallback for every final status.
Precedence == phase = "done" => outcome.key = NoKey \/ outcome.result = "invalid"
                                 \/ outcome.key = MostSpecific(keys, outcome.status)
SelectionMatchesOracle == phase = "done" =>
  \A c \in Codes : ClientSelect(c) = MostSpecific(keys, c)
\* P2: an admitted (branch, status) pair is decoded as that branch at that status.
Agreement == phase = "done" /\ wire.branch # Unmapped /\ bodyOK =>
               outcome = [result |-> "value", key |-> branch, status |-> sent, side |-> kind]
\* P5: a decode failure never reaches a less specific key.
NoFallthrough == phase = "done" /\ outcome.result = "value" =>
                   outcome.key = Select(keys, outcome.status)
\* P5: a designed error is never returned as a success.
SideIntegrity == phase = "done" /\ wire.kind \in {"success", "error"} /\ outcome.result = "value" =>
                   outcome.side = wire.kind
\* P5: a selected key whose body fails, or a body-bearing key at a bodiless
\* status, is a terminal invalid response for that same key.
InvalidBodyRejected ==
  phase = "done" /\ ClientSelect(wire.status) # NoKey /\
  (~bodyOK \/ (body[ClientSelect(wire.status)] /\ Bodiless(wire.status)))
    => outcome.result = "invalid"
\* P7 classification only: authored publication occurs exactly for statuses in
\* the selected branch's runtime domain; anything else is published by the
\* framework error path. Commit timing and header restoration are runtime-test
\* obligations, not properties of this model.
AuthoredPublicationInDomain == phase = "done" =>
  /\ (wire.origin = "authored" => wire.status = sent /\ sent \in Domain(branch))
  /\ (branch = Unmapped \/ sent \notin Domain(branch) => wire.origin = "framework")
\* P7 pinned aliasing: an unmapped failure is returned only through the key that
\* covers 500, and only when that branch decodes the problem body.
UnmappedAliasing == phase = "done" /\ wire.origin = "framework" =>
                      outcome.result = "invalid" \/ outcome.key = Select(keys, 500)
=============================================================================
