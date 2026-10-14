-------------------------- MODULE OperationShutdown --------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS BoundRead, BoundOther, BoundRetry, JoinEvery, SealBeforeLock
VARIABLES phase, owns, requested, closing, completed, returned, blackhole, sealed
vars == <<phase, owns, requested, closing, completed, returned, blackhole, sealed>>
(* replica owns rmap PubSub setup, receive/reconnect and local handle cleanup. *)
Actors == {"reader", "maintenance", "acquisition", "replica"}
Callers == {1, 2}
Active == {"pool", "dial", "write", "read", "retry", "cleanup"}
Init == /\ phase = [a \in Actors |-> "idle"]
        /\ owns = [a \in Actors |-> FALSE]
        /\ requested = {}
        /\ closing = FALSE
        /\ completed = FALSE
        /\ returned = {}
        /\ blackhole \in BOOLEAN
        /\ sealed = {}
Admit(a) == /\ ~closing /\ phase[a] = "idle"
            /\ phase' = [phase EXCEPT ![a] = "pool"]
            /\ UNCHANGED <<owns, requested, closing, completed, returned, blackhole, sealed>>
Call(c) == /\ c \notin requested
           /\ requested' = requested \cup {c}
           /\ UNCHANGED <<phase, owns, closing, completed, returned, blackhole, sealed>>
LockHeld == (phase["maintenance"] \in Active) \/
            (phase["acquisition"] \in Active)
Stop == /\ requested # {} /\ ~closing
        /\ (SealBeforeLock \/ ~LockHeld)
        /\ closing' = TRUE
        /\ sealed' = {a \in Actors: phase[a] # "idle"}
        /\ UNCHANGED <<phase, owns, requested, completed, returned, blackhole>>
(* Native BLOCK bounds only the reader's read phase. Other phases require
   independent correspondence: pool, dial, write, retries and cleanup. *)
(* The adversarial retry phase represents repeated successful commands whose
   CAS loses, not a blocked command. Only a deadline around the whole logical
   operation bounds that repetition; per-command bounds restart each attempt.
   Stuttering at retry abstracts arbitrarily many conflicting revisions. *)
CanReturn(a) == IF phase[a] = "retry"
               THEN ~blackhole \/ BoundRetry
               ELSE ~blackhole \/ BoundOther \/
                    (BoundRead /\ a = "reader" /\ phase[a] = "read")
Advance(a) == /\ phase[a] \in Active /\ CanReturn(a)
              /\ phase' = [phase EXCEPT ![a] =
                  CASE @ = "pool" -> "dial"
                    [] @ = "dial" -> "write"
                    [] @ = "write" -> "read"
                    [] @ = "read" -> "retry"
                    [] @ = "retry" -> IF a \in {"acquisition", "replica"} THEN "cleanup" ELSE "done"
                    [] OTHER -> "done"]
              /\ owns' = [owns EXCEPT ![a] =
                  IF a \in {"acquisition", "replica"} /\ phase[a] = "read" THEN TRUE
                  ELSE IF phase[a] = "cleanup" THEN FALSE ELSE @]
              /\ UNCHANGED <<requested, closing, completed, returned, blackhole, sealed>>
Join == /\ closing /\ ~completed
        /\ \A a \in Actors: phase[a] \notin Active /\ ~owns[a]
        /\ completed' = TRUE
        /\ UNCHANGED <<phase, owns, requested, closing, returned, blackhole, sealed>>
Return(c) == /\ c \in requested \ returned
             /\ (completed \/ (~JoinEvery /\ closing /\ c = 2))
             /\ returned' = returned \cup {c}
             /\ UNCHANGED <<phase, owns, requested, closing, completed, blackhole, sealed>>
Next == (\E a \in Actors: Admit(a) \/ Advance(a)) \/
        (\E c \in Callers: Call(c) \/ Return(c)) \/ Stop \/ Join
Spec == Init /\ [][Next]_vars /\ WF_vars(Stop) /\ WF_vars(Join)
        /\ (\A a \in Actors: WF_vars(Advance(a)))
        /\ (\A c \in Callers: WF_vars(Call(c)) /\ WF_vars(Return(c)))
TypeOK == /\ phase \in [Actors -> Active \cup {"idle", "done"}]
          /\ owns \in [Actors -> BOOLEAN]
          /\ requested \subseteq Callers /\ returned \subseteq requested
          /\ closing \in BOOLEAN /\ completed \in BOOLEAN /\ blackhole \in BOOLEAN
          /\ sealed \subseteq Actors
JoinedOwnsNothing == completed => \A a \in Actors: phase[a] \notin Active /\ ~owns[a]
EveryCloseJoins == returned # {} => completed
AdmissionSealed == closing => {a \in Actors: phase[a] # "idle"} = sealed
Terminates == requested # {} ~> returned = requested
=============================================================================
