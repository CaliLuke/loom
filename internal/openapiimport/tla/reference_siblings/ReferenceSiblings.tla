---------------------- MODULE ReferenceSiblings ----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES base, sibling, declaredRequired, localRequired
SourceTokens == [owner |-> {"null", "valid"}, occurrence |-> {"valid"}]
ReadTarget == SourceTokens[IF Mode = "source" THEN "occurrence" ELSE "owner"]
SourceOwnershipPreserved == ReadTarget = SourceTokens.owner
Values == {"short", "valid", "long", "null"}
Init == /\ base \in SUBSET Values
        /\ sibling \in SUBSET Values
        /\ base \cap sibling # {}
        /\ declaredRequired \in BOOLEAN
        /\ localRequired \in BOOLEAN
Admitted == Mode # "reject" /\ base # {}
Occurrence == IF Mode = "replace" THEN sibling
              ELSE IF Mode = "nullable" THEN (base \cap sibling) \cup {"null"}
              ELSE base \cap sibling
PublishedBase == IF Mode = "mutate" THEN Occurrence ELSE base
SupportedInputAdmitted == Admitted
ConjunctionPreserved == Admitted => Occurrence = base \cap sibling
BaseUnchanged == PublishedBase = base
PublishedReference == IF Mode = "flatten" THEN {} ELSE base
ReferencePreserved == PublishedReference = base
Carrier(required) == IF required THEN "native" ELSE "optional"
RenderedCarrier == Carrier(IF Mode = "storage" THEN declaredRequired \/ localRequired ELSE declaredRequired)
DeclaredStoragePreserved == RenderedCarrier = Carrier(declaredRequired)
Next == UNCHANGED <<base, sibling, declaredRequired, localRequired>>
Spec == Init /\ [][Next]_<<base, sibling, declaredRequired, localRequired>>
======================================================================
