------------------------- MODULE StatusBindings -------------------------
\* #976 response status source: one StatusField shared by status-selected
\* success branches, branch-owned body/header fields, plan admission of
\* competing mappings, and client reconstruction order before validation.
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
VARIABLES extra, v, present, foreign, wire, phase, admitted, result
vars == <<extra, v, present, foreign, wire, phase, admitted, result>>

\* Branches: exact 200 (body "ok", header "etag", both branch-required) and
\* class 2XX (body "accepted", branch-required). StatusField has Minimum(200).
Fields == {"ok", "etag", "accepted"}
Owned(b) == IF b = "200" THEN {"ok", "etag"} ELSE {"accepted"}
Values == {0, 200, 201, 204, 404}
NoStatus == 999
Select(c) == IF c = 200 THEN "200" ELSE IF c >= 200 /\ c <= 299 THEN "2XX" ELSE "none"
Domain(b) == IF b = "200" THEN {200} ELSE {c \in Values : c >= 200 /\ c <= 299 /\ c # 200 /\ c # 204}
\* The checked plan excludes the StatusField from implicit bodies; the
\* implicit-body and body-last candidates leave it there as a body member.
BodyCarriesStatus == extra = "body" \/ Policy \in {"implicit-body", "body-last"}
\* Status field sources written by the client, in reconstruction order:
\* body, headers, then the response status (checked), on every branch.
StatusSources(b) ==
  LET base == IF Policy = "exact-unbound" /\ b = "200" THEN <<>> ELSE <<"status">>
      body == IF BodyCarriesStatus THEN <<"body">> ELSE <<>>
      header == IF extra = "header" THEN <<"header">> ELSE <<>>
  IN IF Policy = "body-last" THEN header \o base \o body ELSE body \o header \o base

Init == /\ extra \in {"none", "header", "body"}
        /\ v \in Values
        /\ present \in SUBSET Fields
        /\ foreign \in BOOLEAN
        /\ wire = [status |-> 0, fields |-> {}, header |-> NoStatus, body |-> NoStatus]
        /\ phase = "plan" /\ admitted = FALSE
        /\ result = [kind |-> "none", branch |-> "none", status |-> 0, fields |-> {}]

\* The shared binding plan rejects a StatusField that is also explicitly
\* mapped to a header or body member.
Plan == /\ phase = "plan"
        /\ admitted' = (extra = "none" \/ Policy = "competing")
        /\ phase' = IF admitted' THEN "encode" ELSE "rejected"
        /\ UNCHANGED <<extra, v, present, foreign, wire, result>>

\* A Loom server selects the branch from the field value and checks the
\* selected branch's domain and branch-required members before commit. A
\* foreign server sends any status, members and competing copies.
Encode == /\ phase = "encode"
          /\ IF foreign
             THEN \E s \in Values \ {0}, fs \in SUBSET Fields, h \in {NoStatus, 404}, bd \in {NoStatus, 404} :
                    wire' = [status |-> s, fields |-> fs, header |-> h, body |-> bd]
             ELSE LET b == Select(v) IN
                  IF b # "none" /\ v \in Domain(b) /\ Owned(b) \subseteq present
                  THEN wire' = [status |-> v, fields |-> Owned(b),
                                header |-> IF extra = "header" THEN v ELSE NoStatus,
                                body |-> IF BodyCarriesStatus THEN v ELSE NoStatus]
                  ELSE wire' = [status |-> 500, fields |-> {}, header |-> NoStatus, body |-> NoStatus]
          /\ phase' = "decode"
          /\ UNCHANGED <<extra, v, present, foreign, admitted, result>>

RECURSIVE Apply(_, _)
Apply(srcs, acc) ==
  IF srcs = <<>> THEN acc
  ELSE LET s == Head(srcs)
           val == CASE s = "status" -> wire.status
                    [] s = "header" -> IF wire.header = NoStatus THEN acc ELSE wire.header
                    [] OTHER -> IF wire.body = NoStatus THEN acc ELSE wire.body
       IN Apply(Tail(srcs), val)

\* The client selects once, reconstructs owned members and the status, then
\* validates. validate-first checks the result before the status is assigned.
Decode == /\ phase = "decode"
          /\ LET b == Select(wire.status)
                 status == IF b = "none" THEN 0 ELSE Apply(StatusSources(b), 0)
                 checked == IF Policy = "validate-first" THEN 0 ELSE status
             IN result' =
                  IF b = "none" THEN [kind |-> "invalid", branch |-> "none", status |-> 0, fields |-> {}]
                  ELSE IF ~(Owned(b) \subseteq wire.fields) \/ checked < 200
                  THEN [kind |-> "invalid", branch |-> b, status |-> status, fields |-> {}]
                  ELSE [kind |-> "value", branch |-> b, status |-> status,
                        fields |-> IF Policy = "copy-all" THEN wire.fields ELSE wire.fields \cap Owned(b)]
          /\ phase' = "done"
          /\ UNCHANGED <<extra, v, present, foreign, wire, admitted>>

Next == Plan \/ Encode \/ Decode \/ (phase \in {"done", "rejected"} /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

\* The reconstructed StatusField equals the actual response status.
StatusIsolation == phase = "done" /\ result.kind = "value" => result.status = wire.status
\* A Loom server's admitted (branch, status) decodes to the same pair.
Agreement == phase = "done" /\ ~foreign /\ wire.status # 500 =>
               result = [kind |-> "value", branch |-> Select(v), status |-> v, fields |-> Owned(Select(v))]
\* The server never commits a selected branch without its required members.
BranchRequired == phase \in {"decode", "done"} /\ ~foreign /\ wire.status # 500 =>
                    Owned(Select(wire.status)) \subseteq wire.fields
\* No bound status source is silently discarded: every source the plan reads
\* either is absent or agrees with the actual status of an accepted response.
SourceValue(src) == CASE src = "status" -> wire.status [] src = "header" -> wire.header [] OTHER -> wire.body
NoDroppedBinding == phase = "done" /\ result.kind = "value" =>
                      \A i \in 1..Len(StatusSources(result.branch)) :
                        SourceValue(StatusSources(result.branch)[i]) \in {wire.status, NoStatus}
\* A Loom server never writes the StatusField as a body member (P6).
NoStatusInBody == phase \in {"decode", "done"} /\ ~foreign => wire.body = NoStatus
\* Members owned by another branch never appear in a decoded result.
BranchDisjoint == phase = "done" /\ result.kind = "value" => result.fields \subseteq Owned(result.branch)
=============================================================================
