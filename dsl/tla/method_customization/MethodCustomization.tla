----------------------- MODULE MethodCustomization -----------------------
EXTENDS Naturals, TLC
CONSTANT Policy
VARIABLES pc, effect, replaced, runs, samples, local, bases, typ, decl
vars == <<pc, effect, replaced, runs, samples, local, bases, typ, decl>>
Effects == {"sample", "required", "extend"}
Intrinsic == {"id"}
Decl(name) == [name |-> name, req |-> Intrinsic, bases |-> 0]
\* "source" is the referenced named type, "other" a shared type a callback may
\* install directly, and "copy" the detached occurrence declaration.
Shared == [source |-> Decl("Value"), other |-> Decl("Other")]
Steps == IF Policy = "two-owner"
         THEN <<"run", "materialize", "run", "extend", "done">>
         ELSE <<"run", "materialize", "done">>
Distinct == effect \in {"required", "extend"}
Init == /\ pc = 1 /\ effect \in Effects /\ replaced \in BOOLEAN
        /\ runs = 0 /\ samples = 0 /\ local = {} /\ bases = 0
        /\ typ = "copy"
        /\ decl = [d \in {"source", "other", "copy"} |->
                    IF d = "copy" THEN Decl("Value") ELSE Shared[d]]
Run == /\ Steps[pc] = "run"
       /\ runs' = runs + 1
       /\ samples' = samples + 1
       /\ local' = IF effect = "required" THEN {"name"} ELSE {}
       /\ bases' = IF effect = "extend" THEN bases + 1 ELSE bases
       /\ typ' = IF replaced THEN "other" ELSE typ
       /\ pc' = pc + 1
       /\ UNCHANGED <<effect, replaced, decl>>
\* two-owner and captured materialize the copy allocated before the callback;
\* two-owner moves bases later. once reads the completed occurrence type and
\* materializes only while that type is still its own copy.
Materialize ==
  /\ Steps[pc] = "materialize"
  /\ LET target == IF Policy = "once" THEN typ ELSE "copy"
         moveBases == Policy # "two-owner"
     IN /\ decl' = IF target = "copy" /\ (local # {} \/ bases > 0)
                   THEN [decl EXCEPT !["copy"] =
                           [name |-> "Value_m",
                            req |-> @.req \cup local,
                            bases |-> IF moveBases THEN @.bases + bases ELSE @.bases]]
                   ELSE decl
        /\ bases' = IF moveBases /\ target = "copy" THEN 0 ELSE bases
  /\ pc' = pc + 1
  /\ UNCHANGED <<effect, replaced, runs, samples, local, typ>>
\* two-owner: after the second run, bases move onto the current occurrence type.
Extend == /\ Steps[pc] = "extend"
          /\ decl' = IF bases > 0
                     THEN [decl EXCEPT ![typ] =
                             [name |-> IF typ = "copy" THEN "Value_m" ELSE @.name,
                              req |-> @.req, bases |-> @.bases + bases]]
                     ELSE decl
          /\ bases' = 0
          /\ pc' = pc + 1
          /\ UNCHANGED <<effect, replaced, runs, samples, local, typ>>
Next == Run \/ Materialize \/ Extend \/ (Steps[pc] = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars

Done == Steps[pc] = "done"
Materialized == \E i \in 1..(pc - 1): Steps[i] = "materialize"
\* Materialization consumes completed callback facts; nothing runs after it.
OnceBeforeMaterialization == Materialized => runs = 1
ExactlyOnce == Done => runs = 1 /\ samples = 1
\* Shared declarations are never renamed, extended or given local requirements.
SourceIsolated == decl["source"] = Shared.source /\ decl["other"] = Shared.other
\* A copy the occurrence no longer uses is never materialized.
NoStaleMaterialization == typ # "copy" => decl["copy"] = Decl("Value")
\* An owned copy gets its own identity exactly when it changes the declaration,
\* and owns each authored requirement and Extend contributor once.
OwnedDeclaration ==
  Done /\ typ = "copy" =>
    decl["copy"] = [name |-> IF Distinct THEN "Value_m" ELSE "Value",
                    req |-> Intrinsic \cup local,
                    bases |-> IF effect = "extend" THEN 1 ELSE 0]
=============================================================================
