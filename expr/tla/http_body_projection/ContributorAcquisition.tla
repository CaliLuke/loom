---------------------- MODULE ContributorAcquisition ----------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Mode
VARIABLES ordering, position, memo, acquired, outputs
vars == <<ordering, position, memo, acquired, outputs>>

(* A Parent's authored local schema is equal in every case. Child-only thumb
   selection changes its publication closure. The same authored count=0
   example projects differently under captured request pointer/presence policy
   and native response omitzero policy. These policies come from preparation,
   not from schema attribute shape. Bound layouts are already resolved inputs.
   Two sequential caches model the actual contributor memo and early named
   schema acquisition; neither is the completed-graph equivalence registry. *)
O(codec, excluded, policy) == [codec |-> codec, excluded |-> excluded, policy |-> policy]
Full == O("form", {}, "request")
Removed == O("form", {"thumb"}, "request")
Request == O("json", {"id"}, "request")
Response == O("json", {"id"}, "response")
FullRequest == O("json", {}, "request")
FullResponse == O("json", {}, "response")
Orders == {<<Full, Removed>>, <<Removed, Full>>, <<Request, Response>>, <<Response, Request>>, <<Full, Full>>, <<FullRequest, FullResponse>>, <<FullResponse, FullRequest>>}
PolicyContext(o) == IF Mode = "hardcodedRequest" \/ (Mode = "canonicalShortcut" /\ o.codec = "json" /\ o.excluded = {}) THEN "request" ELSE o.policy
Example(policy) == IF policy = "request" THEN {"count=0"} ELSE {}
Materialized(o) ==
  [parent |-> [members |-> {"kind", "count", "id"} \ o.excluded,
               example |-> Example(PolicyContext(o))],
   published |-> [members |-> {"kind", "count", "id", "thumb"} \ o.excluded,
                  example |-> Example(PolicyContext(o))]]
Expected(o) ==
  [parent |-> [members |-> {"kind", "count", "id"} \ o.excluded,
               example |-> Example(o.policy)],
   published |-> [members |-> {"kind", "count", "id", "thumb"} \ o.excluded,
                  example |-> Example(o.policy)]]
MemoKey(o) == <<o.codec, o.excluded, IF Mode = "policyMemoOmitted" THEN "any" ELSE o.policy>>
AcquisitionKey(o) ==
  CASE Mode = "canonicalShortcut" /\ o.codec = "json" /\ o.excluded = {} -> <<"canonical">>
    [] Mode = "localAcquisition" -> <<o.codec, o.excluded \ {"thumb"}, "authored count=0">>
    [] Mode = "policyAcquisitionOmitted" -> <<o.codec, o.excluded, "any">>
    [] Mode = "selectionAcquisitionOmitted" -> <<o.codec, o.excluded \ {"thumb"}, o.policy>>
    [] OTHER -> <<o.codec, o.excluded, o.policy>>
Find(cache, key) == CHOOSE pair \in cache: pair[1] = key
Has(cache, key) == \E pair \in cache: pair[1] = key

Init == ordering \in Orders /\ position = 1 /\ memo = {} /\ acquired = {} /\ outputs = <<>>
Next ==
  IF position <= Len(ordering)
  THEN LET o == ordering[position]
           mkey == MemoKey(o)
           akey == AcquisitionKey(o)
           built == IF Has(memo, mkey) THEN Find(memo, mkey)[2] ELSE Materialized(o)
           reused == IF Has(acquired, akey) THEN Find(acquired, akey)[2] ELSE built
       IN /\ memo' = IF Has(memo, mkey) THEN memo ELSE memo \cup {<<mkey, built>>}
          /\ acquired' = IF Has(acquired, akey) THEN acquired ELSE acquired \cup {<<akey, reused>>}
          /\ outputs' = Append(outputs, reused)
          /\ position' = position + 1
          /\ UNCHANGED ordering
  ELSE UNCHANGED vars
Spec == Init /\ [][Next]_vars

TypeOK == ordering \in Orders /\ position \in 1..3 /\ Len(outputs) = position - 1
OwnMaterialization == \A i \in 1..Len(outputs): outputs[i] = Expected(ordering[i])
(* Comparison uses expected complete publication, not a possibly already lost
   context or only locally equal Parent properties. *)
AcquisitionOwnContext ==
  \A i, j \in 1..Len(outputs): AcquisitionKey(ordering[i]) = AcquisitionKey(ordering[j]) => Expected(ordering[i]) = Expected(ordering[j])
MemoOwnContext ==
  \A i, j \in 1..Len(outputs): MemoKey(ordering[i]) = MemoKey(ordering[j]) => Expected(ordering[i]) = Expected(ordering[j])
(* The final registry may merge exactly equal completed observable graphs.
   This is separate from provisional context keys and never merges only by
   local Parent shape. Full/Full demonstrates the non-vacuous equal case. *)
CompletedEquivalence ==
  \A i, j \in 1..Len(outputs): outputs[i] = outputs[j] => Expected(ordering[i]) = Expected(ordering[j])
=============================================================================
