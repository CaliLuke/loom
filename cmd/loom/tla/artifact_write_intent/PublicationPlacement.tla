----------------------- MODULE PublicationPlacement -----------------------
EXTENDS Naturals, TLC
CONSTANTS Mode, ParentWritable
VARIABLES phase, candidates, live, temporary
vars == <<phase, candidates, live, temporary>>
Units == {"gen", "file"}
DestFS == [u \in Units |-> IF u = "gen" THEN 1 ELSE 2]
Old == [u \in Units |-> "old"]
Init == /\ phase = "render" /\ candidates = [u \in Units |-> 0]
        /\ live = Old /\ temporary = {}
Render == /\ phase = "render"
          /\ phase' = IF Mode = "current" /\ ~ParentWritable
                       THEN "parent-denied" ELSE "prepare"
          /\ temporary' = IF phase' = "prepare" THEN {"render"} ELSE {}
          /\ UNCHANGED <<candidates, live>>
Prepare == /\ phase = "prepare" /\ phase' = "gen"
           /\ candidates' = IF Mode = "prepared" THEN DestFS
                            ELSE [u \in Units |-> IF Mode = "current" THEN 0 ELSE 1]
           /\ temporary' = temporary \cup Units /\ UNCHANGED live
PrepareFailure == /\ phase = "prepare" /\ phase' = "failed"
                  /\ temporary' = {} /\ UNCHANGED <<candidates, live>>
PublishGen == /\ phase = "gen" /\ phase' = "file"
              /\ live' = [live EXCEPT !["gen"] = "new"]
              /\ temporary' = temporary \ {"gen"}
              /\ UNCHANGED candidates
PublishFile == /\ phase = "file" /\ phase' = "finish"
               /\ live' = [live EXCEPT !["file"] = "new"]
               /\ temporary' = temporary \ {"file"}
               /\ UNCHANGED candidates
Rollback == /\ phase \in {"file", "finish"} /\ phase' = "failed"
            /\ live' = Old /\ temporary' = {} /\ UNCHANGED candidates
Finish == /\ phase = "finish" /\ phase' = "done" /\ temporary' = {}
          /\ UNCHANGED <<candidates, live>>
Next == Render \/ Prepare \/ PrepareFailure \/ PublishGen \/ PublishFile \/ Rollback \/ Finish
Spec == Init /\ [][Next]_vars
WritableOutputAdmitted == phase # "parent-denied"
PublicationLocal == /\ (phase = "gen" => candidates["gen"] = DestFS["gen"])
                    /\ (phase = "file" => candidates["file"] = DestFS["file"])
FailurePreservesLive == phase = "failed" => live = Old
TerminalClean == phase \in {"failed", "done", "parent-denied"} => temporary = {}
=============================================================================
