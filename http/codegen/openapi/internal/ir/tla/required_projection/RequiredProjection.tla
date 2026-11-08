---- MODULE RequiredProjection ----
EXTENDS FiniteSets
CONSTANT Mode
VARIABLE scenario, emitted
vars == <<scenario, emitted>>
Cases == {"upload", "row", "overlay", "additional", "typedProjection", "usageProjection", "plain"}
Members(c) == CASE c = "upload" -> {"file", "filename"}
                  [] c = "row" -> {"extra", "id"}
                  [] c = "overlay" -> {"extra", "id"}
                  [] c = "additional" -> {"id"}
                  [] c = "typedProjection" -> {"count"}
                  [] c = "usageProjection" -> {"extra"}
                  [] OTHER -> {"id"}
Local(c) == CASE c = "upload" -> {"id"}
                [] c = "row" -> {"extra"}
                [] c = "overlay" -> {"extra"}
                [] c = "additional" -> {"id", "literal"}
                [] c = "typedProjection" -> {"count"}
                [] c = "usageProjection" -> {"extra", "id:literal"}
                [] OTHER -> {}
Parents(c) == CASE c = "upload" -> {"file", "filename"}
                  [] c \in {"row", "overlay", "typedProjection", "usageProjection"} -> {"id"}
                  [] OTHER -> {}
Expected(c) == IF c = "overlay" THEN Local(c)
               ELSE IF c = "additional" THEN Local(c) \cup Parents(c)
               ELSE IF c = "usageProjection" THEN (Local(c) \cup Parents(c)) \ {"id"}
               ELSE (Local(c) \cup Parents(c)) \cap Members(c)
Init == scenario = "pending" /\ emitted = {}
Next == scenario = "pending" /\ \E c \in Cases:
          scenario' = c /\ emitted' =
            IF Mode = "local" THEN Local(c)
            ELSE IF Mode = "all" THEN Local(c) \cup Parents(c)
            ELSE IF Mode = "usageLost" /\ c = "usageProjection" THEN Local(c) \cup Parents(c)
            ELSE Expected(c)
Contract == scenario = "pending" \/ emitted = Expected(scenario)
Spec == Init /\ [][Next]_vars
====
