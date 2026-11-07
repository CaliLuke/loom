-------------------- MODULE MetadataPrecedence --------------------
EXTENDS Naturals
CONSTANT ExplicitFirst
VARIABLE authored, fileCount, header, value, done
vars == <<authored, fileCount, header, value, done>>
Init == /\ authored \in {"absent", "", "explicit"}
        /\ fileCount \in 0..2 /\ header \in {"", "derived"}
        /\ value = "unset" /\ done = FALSE
Derived == IF fileCount = 1 /\ header # "" THEN header ELSE "absent"
Next == /\ ~done
        /\ value' = IF ExplicitFirst /\ authored # "absent" THEN authored
                     ELSE IF Derived # "absent" THEN Derived ELSE authored
        /\ done' = TRUE /\ UNCHANGED <<authored, fileCount, header>>
Spec == Init /\ [][Next]_vars
Precedence == done => value = IF authored # "absent" THEN authored ELSE Derived
=============================================================================
