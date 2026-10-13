---------------------- MODULE ResponseMetadata ----------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES phase, header, wire, captured, safe
vars == <<phase, header, wire, captured, safe>>
Init == /\ phase = "prepare"
        /\ header \in {"json", "plain"}
        /\ wire = "none"
        /\ captured = "none"
        /\ safe = TRUE
Informational == /\ phase = "prepare"
                 /\ header' \in {"json", "plain"}
                 /\ phase' = "ready"
                 /\ UNCHANGED <<wire, captured, safe>>
Commit == /\ phase = "ready"
          /\ wire' = header
          /\ captured' = header
          /\ phase' = "committed"
          /\ UNCHANGED <<header, safe>>
OpaqueCopy == /\ phase = "ready"
              /\ wire' = "json"
              /\ captured' = IF Mode = "FirstRead" THEN header ELSE "unknown"
              /\ header' = "plain"
              /\ phase' = "committed"
              /\ UNCHANGED safe
ForwardedCommit == /\ phase = "ready"
                   /\ wire' = "json"
                   /\ header' = IF header = "json" THEN "plain" ELSE "json"
                   /\ captured' = IF Mode = "AfterCall" THEN header' ELSE "unknown"
                   /\ phase' = "committed"
                   /\ UNCHANGED safe
Mutate == /\ phase = "committed"
          /\ header' \in {"json", "plain"}
          /\ phase' = "observe"
          /\ UNCHANGED <<wire, captured, safe>>
Observe == /\ phase = "observe"
           /\ safe' = ((wire # "json") \/
                 (IF Mode = "Late" THEN header = "json"
                  ELSE captured = "json" \/ captured = "unknown"))
           /\ phase' = "done"
           /\ UNCHANGED <<header, wire, captured>>
Next == Informational \/ Commit \/ OpaqueCopy \/ ForwardedCommit \/ Mutate \/ Observe
Spec == Init /\ [][Next]_vars
NoDisclosure == safe
=============================================================================
