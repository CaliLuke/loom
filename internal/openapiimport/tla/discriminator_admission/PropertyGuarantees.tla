----------------------- MODULE PropertyGuarantees -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES nullable, occurrenceNull, narrowed, required, outerNull,
          version, fallback, phase, outcome, conditional
vars == <<nullable, occurrenceNull, narrowed, required, outerNull,
          version, fallback, phase, outcome, conditional>>

\* Finite value domain for one branch; the other union branch is a required,
\* nonnullable object. A reference occurrence can add null after an object
\* intersection has removed it. An outer wrapper does not own the inner hint.
Values == {"null", "present", "absent"}
AdmitsNull == (nullable /\ ~narrowed) \/ occurrenceNull
Accepted == {v \in Values : (v = "null" /\ AdmitsNull) \/
                              (v = "present") \/ (v = "absent" /\ ~required)}
RequiredKeyword == \A v \in Accepted : v = "null" \/ v = "present"
PropertyPresent == \A v \in Accepted : v = "present"
Optional == IF Mode = "old" THEN ~required
            ELSE ~required \/ AdmitsNull \/ (Mode = "wrapper" /\ outerNull)
Init == /\ nullable \in BOOLEAN /\ occurrenceNull \in BOOLEAN
        /\ narrowed \in BOOLEAN /\ required \in BOOLEAN /\ outerNull \in BOOLEAN
        /\ version \in {30, 31, 32} /\ fallback \in BOOLEAN
        /\ phase = "inspect" /\ outcome = "none" /\ conditional = FALSE
Decide == /\ phase = "inspect"
          /\ conditional' = (required /\ (Mode # "conflate" \/ ~AdmitsNull))
          /\ outcome' = IF (version # 32 /\ fallback) \/
                            (version = 30 /\ Optional) \/
                            (version = 32 /\ Optional /\ ~fallback)
                         THEN "reject" ELSE "admit"
          /\ phase' = "done"
          /\ UNCHANGED <<nullable, occurrenceNull, narrowed, required, outerNull, version, fallback>>
Next == Decide
Spec == Init /\ [][Next]_vars
KeywordPreserved == phase = "done" => conditional = RequiredKeyword
VersionAdmission == phase = "done" =>
  (outcome = "admit") =
    (~(version # 32 /\ fallback) /\
     ~(version = 30 /\ ~PropertyPresent) /\
     ~(version = 32 /\ ~PropertyPresent /\ ~fallback))
=============================================================================
