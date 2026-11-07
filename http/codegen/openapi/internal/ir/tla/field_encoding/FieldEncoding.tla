---------------- MODULE FieldEncoding ----------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Policy
VARIABLE scenario
vars == <<scenario>>
Cases == {"source-array", "absent-singleton", "heterogeneous-items", "conflicting-authored"}
Init == scenario \in Cases
Next == UNCHANGED vars
SourceType == IF scenario = "absent-singleton" THEN "absent" ELSE "array"
NormalizedType == IF scenario = "source-array" THEN "absent" ELSE SourceType
ExportType == IF Policy = "normalized" THEN
                IF scenario = "absent-singleton" THEN "array" ELSE NormalizedType
              ELSE SourceType
Application(typ) == IF typ = "array" THEN "items" ELSE "value"
ItemDefaults == IF scenario = "heterogeneous-items" THEN <<"text", "json">> ELSE <<"text", "text">>
Authored == IF scenario = "conflicting-authored" THEN <<"text", "json">> ELSE <<"default", "default">>
Codec(label, index) == IF label = "default" THEN ItemDefaults[index] ELSE label
Expected(index) == Codec(Authored[index], index)
Candidates == {Authored[i] : i \in DOMAIN Authored} \cup {"default"}
Equivalent(label) == \A i \in DOMAIN Authored : Codec(label, i) = Expected(i)
Compatible == {label \in Candidates : Equivalent(label)}
Rejected == IF Policy = "checked" THEN Compatible = {} ELSE FALSE
Selected == IF Policy = "global" THEN "json"
            ELSE IF Compatible = {} THEN "default"
            ELSE CHOOSE label \in Compatible : TRUE
ApplicationPreserved == ~Rejected => Application(ExportType) = Application(SourceType)
DefaultsPreserved == ~Rejected => Equivalent(Selected)
ConflictReported == (Compatible = {}) => Rejected
Spec == Init /\ [][Next]_vars
======================================================
