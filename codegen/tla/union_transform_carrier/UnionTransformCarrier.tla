---------------- MODULE UnionTransformCarrier ----------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Shapes == {"object", "array", "map", "union", "primitive"}
Nullability == {"intrinsic", "occurrence", "ordinary"}
Presence == {"absent", "null", "value"}
Cases == {c \in [shape: Shapes, nullable: Nullability, presence: Presence]:
          c.nullable # "ordinary" \/ c.presence = "value"}
UsesHelper(c) == c.shape = "object"
Wrapped(c) == c.nullable # "ordinary"
Bypass(c) == UsesHelper(c) /\ Wrapped(c) /\ Mode \in {"old", "names-only"}
VARIABLES selected, done, identityOK, carrierOK, preserved
vars == <<selected, done, identityOK, carrierOK, preserved>>
Init == /\ selected \in Cases /\ done = FALSE
        /\ identityOK = TRUE /\ carrierOK = TRUE /\ preserved = TRUE
Convert ==
    /\ ~done
    /\ done' = TRUE
    /\ UNCHANGED selected
    /\ identityOK' = ~(Bypass(selected) /\ Mode = "old")
    /\ carrierOK' = ~Bypass(selected)
    /\ preserved' = ~(Mode = "drop-null" /\ Wrapped(selected) /\ selected.presence = "null")
Next == Convert \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
CallDefinitionIdentity == identityOK
CallCarrierAgreement == carrierOK
PresencePreserved == preserved
Terminates == <>done
================================================================
