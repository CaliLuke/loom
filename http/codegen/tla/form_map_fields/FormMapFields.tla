---------------- MODULE FormMapFields ----------------
EXTENDS Naturals
CONSTANT Policy
VARIABLE shape, owner
vars == <<shape,owner>>
Init == /\ shape \in {"scalar", "strings", "object", "objects", "nested-array", "array-union"}
        /\ owner \in {"request", "media", "union"}
Next == UNCHANGED vars
Captured == CASE shape = "scalar" -> "text-value"
              [] shape = "strings" -> "text-items"
              [] shape = "object" -> "json-value"
              [] shape = "objects" \/ shape = "nested-array" -> "json-items"
              [] OTHER -> "branch-items"
Reflection == IF shape = "scalar" THEN "text-value" ELSE
              IF shape = "object" THEN "json-value" ELSE "text-items"
Emitted == IF Policy = "old" THEN Reflection ELSE Captured
CodecAgreement == Emitted = Captured
Spec == Init /\ [][Next]_vars
======================================================
