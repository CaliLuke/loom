-------------------------- MODULE MultipartBudget --------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES specimen, phase, readBytes, rejected, dispatched
vars == <<specimen, phase, readBytes, rejected, dispatched>>
Cases == {"exact", "overhead", "epilogue", "source-error"}
Limit == 2
Prefix == IF specimen = "overhead" THEN 3 ELSE 2
Total == IF specimen \in {"overhead", "epilogue"} THEN 3 ELSE 2
SourceError == specimen = "source-error"
Init == /\ specimen \in Cases
        /\ phase = "read"
        /\ readBytes = 0
        /\ rejected = FALSE
        /\ dispatched = FALSE
Read == /\ phase = "read"
        /\ readBytes' = Prefix
        /\ rejected' = (Mode # "values" /\ Prefix > Limit)
        /\ phase' = "callback-success"
        /\ UNCHANGED <<specimen, dispatched>>
Complete == /\ phase = "callback-success"
            /\ phase' = "dispatch"
            /\ readBytes' = IF Mode = "owned" /\ ~rejected THEN Total ELSE readBytes
            /\ rejected' = (rejected \/ (Mode = "owned" /\ (Total > Limit \/ SourceError)))
            /\ UNCHANGED <<specimen, dispatched>>
Dispatch == /\ phase = "dispatch"
            /\ phase' = "done"
            /\ dispatched' = ~rejected
            /\ UNCHANGED <<specimen, readBytes, rejected>>
Done == /\ phase = "done" /\ UNCHANGED vars
Next == Read \/ Complete \/ Dispatch \/ Done
TypeOK == /\ specimen \in Cases /\ phase \in {"read", "callback-success", "dispatch", "done"}
          /\ readBytes \in 0..3 /\ rejected \in BOOLEAN /\ dispatched \in BOOLEAN
Admission == dispatched => (Total <= Limit /\ ~SourceError)
ExactAccepted == (phase = "done" /\ specimen = "exact") => dispatched
=============================================================================
