------------------------- MODULE BackupRecovery -------------------------
EXTENDS FiniteSets, TLC
CONSTANT Mode
Kinds == {"gen", "external"}
VARIABLES phase, moved, backups, liveOriginal, restored, committed, reported
vars == <<phase, moved, backups, liveOriginal, restored, committed, reported>>
Init == /\ phase = "install"
        /\ moved = {} /\ backups = {} /\ liveOriginal = Kinds
        /\ restored = {} /\ committed = FALSE /\ reported = {}
Preserve(k) == /\ phase = "install" /\ k \notin moved
               /\ moved' = moved \cup {k}
               /\ backups' = backups \cup {k}
               /\ liveOriginal' = liveOriginal \ {k}
               /\ UNCHANGED <<phase, restored, committed, reported>>
Commit == /\ phase = "install" /\ moved = Kinds
          /\ phase' = "cleanup" /\ committed' = TRUE
          /\ UNCHANGED <<moved, backups, liveOriginal, restored, reported>>
Fail == /\ phase = "install" /\ phase' = "rollback"
        /\ UNCHANGED <<moved, backups, liveOriginal, restored, committed, reported>>
Restore(k, success) == /\ phase = "rollback" /\ k \in moved \ restored
                      /\ restored' = restored \cup {k}
                      /\ backups' = (IF success THEN backups \ {k} ELSE backups)
                      /\ liveOriginal' = (IF success THEN liveOriginal \cup {k} ELSE liveOriginal)
                      /\ UNCHANGED <<phase, moved, committed, reported>>
FinishRollback == /\ phase = "rollback" /\ restored = moved
                  /\ phase' = "cleanup"
                  /\ UNCHANGED <<moved, backups, liveOriginal, restored, committed, reported>>
Cleanup == /\ phase \in {"cleanup", "done"}
           /\ LET retained == IF committed THEN {}
                              ELSE CASE Mode = "current" -> {}
                                        [] Mode = "external-only" -> backups \cap {"external"}
                                        [] OTHER -> backups
              IN /\ backups' = retained
                 /\ reported' = retained
           /\ phase' = "done"
           /\ UNCHANGED <<moved, liveOriginal, restored, committed>>
Next == (\E k \in Kinds: Preserve(k)) \/ Commit \/ Fail
        \/ (\E k \in Kinds, success \in BOOLEAN: Restore(k, success))
        \/ FinishRollback \/ Cleanup
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in {"install", "rollback", "cleanup", "done"}
          /\ moved \subseteq Kinds /\ backups \subseteq moved
          /\ liveOriginal \subseteq Kinds /\ restored \subseteq moved
          /\ committed \in BOOLEAN /\ reported \subseteq Kinds
RecoverableOriginals == ~committed => liveOriginal \cup backups = Kinds
RecoveryReported == phase = "done" /\ ~committed => backups \subseteq reported
CommittedCleanup == phase = "done" /\ committed => backups = {}
RestoredNeedsNoBackup == phase = "done" /\ ~committed /\ liveOriginal = Kinds => backups = {}
=============================================================================
