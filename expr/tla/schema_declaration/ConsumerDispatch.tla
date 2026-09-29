-------------------------- MODULE ConsumerDispatch --------------------------
EXTENDS Naturals, Sequences
CONSTANT Policy
VARIABLES roles, selectedView, descriptions, cursor, phase, owner, working,
          memo, acquired, baseConstraints, useConstraints, sourceNullable, usageNullable
vars == <<roles, selectedView, descriptions, cursor, phase, owner, working,
          memo, acquired, baseConstraints, useConstraints, sourceNullable, usageNullable>>

\* Positions abstract a named object, a named collection element, and a named
\* nested object. Their extraction and wrapper traversal are input assumptions.
Positions == {"object", "collection-element", "nested-object"}
Roles == {"async-inline", "ordinary-response", "retained-cut"}
Fields == {"a", "b", "c"}
EnumValues == {1, 2, 3}
ConstructorPolicies ==
    {"owner-first-ordinary-validation", "owner-first-ordinary-nullability", "consumer-owned"}
Requests == <<"object", "collection-element", "nested-object",
              "object", "collection-element", "nested-object">>
Input(position) == [fields |-> Fields, description |-> descriptions[position]]
OrdinaryView(value) ==
    [fields |-> value.fields \cap selectedView,
     description |-> IF value.description = "absent"
                     THEN "generated-view" ELSE value.description]
Current == Requests[cursor]
CurrentOwner == IF roles[Current] = "async-inline" THEN "inline" ELSE "ordinary"

Init ==
    /\ roles \in [Positions -> Roles]
    /\ selectedView \in (SUBSET Fields) \ {{}}
    /\ descriptions \in [Positions -> {"absent", "authored"}]
    \* Preserve the original view-only experiments with neutral constructor
    \* inputs. Strengthened experiments cover every nonempty enum pair and
    \* both independent nullability flags, shared across the three positions.
    /\ baseConstraints \in IF Policy \in ConstructorPolicies
                           THEN (SUBSET EnumValues) \ {{}} ELSE {EnumValues}
    /\ useConstraints \in IF Policy \in ConstructorPolicies
                          THEN (SUBSET EnumValues) \ {{}} ELSE {EnumValues}
    /\ sourceNullable \in IF Policy \in ConstructorPolicies THEN BOOLEAN ELSE {FALSE}
    /\ usageNullable \in IF Policy \in ConstructorPolicies THEN BOOLEAN ELSE {FALSE}
    /\ cursor = 1
    /\ phase = "capture"
    /\ owner = "unselected"
    /\ working = [fields |-> {}, description |-> "absent"]
    /\ memo = <<>>
    /\ acquired = <<>>

Capture ==
    /\ phase = "capture" /\ cursor <= Len(Requests)
    /\ working' = Input(Current)
    /\ owner' = "unselected"
    /\ phase' = "first"
    /\ UNCHANGED <<roles, selectedView, descriptions, cursor, memo, acquired,
                    baseConstraints, useConstraints, sourceNullable, usageNullable>>

\* The defect applies ordinary view policy while the consumer is undecided.
\* The candidate selects the consumer before applying any view policy.
First ==
    /\ phase = "first"
    /\ working' = IF Policy = "ordinary-first" THEN OrdinaryView(working) ELSE working
    /\ owner' = IF Policy = "ordinary-first" THEN owner ELSE CurrentOwner
    /\ phase' = "second"
    /\ UNCHANGED <<roles, selectedView, descriptions, cursor, memo, acquired,
                    baseConstraints, useConstraints, sourceNullable, usageNullable>>
Second ==
    /\ phase = "second"
    /\ owner' = IF Policy = "ordinary-first" THEN CurrentOwner ELSE owner
    /\ working' = IF Policy \in ConstructorPolicies \cup {"owner-first"}
                     /\ owner = "ordinary"
                  THEN OrdinaryView(working) ELSE working
    /\ phase' = "acquire"
    /\ UNCHANGED <<roles, selectedView, descriptions, cursor, memo, acquired,
                    baseConstraints, useConstraints, sourceNullable, usageNullable>>

\* These are finite baseline policies, not byte projection or a proposal to
\* unify consumer semantics. A correctly dispatched inline constructor can
\* still import ordinary alias validation/nullability and change the contract.
Constructed ==
    [fields |-> working.fields, description |-> working.description,
     allowed |-> IF owner = "inline" /\
                    Policy \in {"consumer-owned", "owner-first-ordinary-nullability"}
                 THEN useConstraints ELSE baseConstraints \cap useConstraints,
     nullable |-> IF owner = "inline" /\
                     Policy \in {"consumer-owned", "owner-first-ordinary-validation"}
                  THEN sourceNullable ELSE sourceNullable \/ usageNullable]

\* All policies use the same correctly occurrence-and-usage-owned cache.
\* A cache miss trusts the supplied working shape. Correct identity alone
\* cannot recover fields/annotations already changed by the wrong consumer.
Matches == {i \in 1..Len(memo) :
               memo[i].position = Current /\ memo[i].role = roles[Current]}
FirstIndex(indices) == CHOOSE i \in indices : \A j \in indices : i <= j
Acquire ==
    /\ phase = "acquire"
    /\ LET id == IF Matches = {} THEN Len(memo) + 1 ELSE FirstIndex(Matches)
           entry == IF Matches = {}
                    THEN [position |-> Current, role |-> roles[Current], value |-> Constructed]
                    ELSE memo[id]
       IN /\ memo' = IF Matches = {} THEN Append(memo, entry) ELSE memo
          /\ acquired' = Append(acquired,
                  [position |-> Current, role |-> roles[Current],
                   value |-> entry.value, id |-> id])
    /\ cursor' = cursor + 1
    /\ phase' = IF cursor = Len(Requests) THEN "done" ELSE "capture"
    /\ UNCHANGED <<roles, selectedView, descriptions, owner, working,
                    baseConstraints, useConstraints, sourceNullable, usageNullable>>
Next == Capture \/ First \/ Second \/ Acquire
Spec == Init /\ [][Next]_vars

\* Consumer contract comes from original inputs, never the working or cached
\* result. Each assertion covers every acquired position, including repeats.
InlineShapeAuthority ==
    \A i \in 1..Len(acquired) :
        acquired[i].role = "async-inline" => acquired[i].value.fields = Fields
InlineAnnotationAuthority ==
    \A i \in 1..Len(acquired) :
        acquired[i].role = "async-inline" =>
          acquired[i].value.description = descriptions[acquired[i].position]
OrdinaryAndRetainedPolicy ==
    \A i \in 1..Len(acquired) :
        acquired[i].role \in {"ordinary-response", "retained-cut"} =>
          /\ acquired[i].value.fields = selectedView
          /\ acquired[i].value.description =
               IF descriptions[acquired[i].position] = "absent"
               THEN "generated-view" ELSE descriptions[acquired[i].position]
ConsumerConstraintAuthority ==
    \A i \in 1..Len(acquired) :
        acquired[i].value.allowed =
          IF acquired[i].role = "async-inline"
          THEN useConstraints ELSE baseConstraints \cap useConstraints
ConsumerNullabilityAuthority ==
    \A i \in 1..Len(acquired) :
        acquired[i].value.nullable =
          IF acquired[i].role = "async-inline"
          THEN sourceNullable ELSE sourceNullable \/ usageNullable
RepeatedOccurrenceReused ==
    \A i, j \in 1..Len(acquired) :
        acquired[i].position = acquired[j].position => acquired[i].id = acquired[j].id
MemoIdentityStable ==
    \A i \in 1..Len(acquired) :
        acquired[i].id \in 1..Len(memo) /\
        memo[acquired[i].id].value = acquired[i].value
=============================================================================
