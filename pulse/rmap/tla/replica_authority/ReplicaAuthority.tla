------------------------- MODULE ReplicaAuthority -------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Policy
VARIABLES epoch, rev, content, writes, messages, localEpoch, localRev,
          localContent, refresh, stopped, targetEpoch, targetRev,
          completed, badClaim, badAck
vars == <<epoch, rev, content, writes, messages, localEpoch, localRev,
          localContent, refresh, stopped, targetEpoch, targetRev,
          completed, badClaim, badAck>>
Init == /\ epoch = 1 /\ rev = 0 /\ content = {} /\ writes = 0
        /\ messages = <<>> /\ localEpoch = 1 /\ localRev = 0
        /\ localContent = {} /\ refresh = FALSE /\ stopped = FALSE
        /\ targetEpoch = 0 /\ targetRev = 0 /\ completed = FALSE
        /\ badClaim = FALSE /\ badAck = FALSE
Write == /\ ~stopped /\ writes < 3
         /\ writes' = writes + 1 /\ rev' = rev + 1
         /\ content' = content \cup {writes + 1}
         /\ messages' = Append(messages, [generation |-> epoch, sequence |-> rev + 1,
                                         key |-> writes + 1, snapshot |-> content'])
         /\ targetEpoch' = IF writes = 2 THEN epoch ELSE targetEpoch
         /\ targetRev' = IF writes = 2 THEN rev + 1 ELSE targetRev
         /\ UNCHANGED <<epoch, localEpoch, localRev, localContent, refresh,
                         stopped, completed, badClaim, badAck>>
Expire == /\ ~stopped /\ epoch = 1
          /\ epoch' = 2 /\ rev' = 0 /\ content' = {}
          /\ UNCHANGED <<writes, messages, localEpoch, localRev, localContent,
                          refresh, stopped, targetEpoch, targetRev, completed,
                          badClaim, badAck>>
Drop == /\ Len(messages) > 0 /\ messages' = Tail(messages)
        /\ UNCHANGED <<epoch, rev, content, writes, localEpoch, localRev,
                        localContent, refresh, stopped, targetEpoch, targetRev,
                        completed, badClaim, badAck>>
Deliver == /\ Len(messages) > 0
           /\ LET msg == Head(messages)
                  generationMismatch == msg.generation # localEpoch
                  gap == msg.sequence > localRev + 1
                  repair == (Policy \in {"GenerationOnly", "Triggered", "Owned"} /\ generationMismatch)
                            \/ (Policy \in {"GapOnly", "Triggered", "Owned"} /\ gap)
                  accept == ~repair /\ msg.sequence > localRev
                  updated == localContent \cup {msg.key}
              IN /\ messages' = Tail(messages)
                 /\ refresh' = (refresh \/ repair)
                 /\ localEpoch' = IF accept THEN msg.generation ELSE localEpoch
                 /\ localRev' = IF accept THEN msg.sequence ELSE localRev
                 /\ localContent' = IF accept THEN updated ELSE localContent
                 /\ badClaim' = (badClaim \/ (accept /\ updated # msg.snapshot))
           /\ UNCHANGED <<epoch, rev, content, writes, stopped, targetEpoch,
                           targetRev, completed, badAck>>
Snapshot == /\ refresh \/ (Policy = "Owned" /\
                            <<localEpoch,localRev,localContent>> # <<epoch,rev,content>>)
            /\ localEpoch' = epoch /\ localRev' = rev /\ localContent' = content
            /\ refresh' = FALSE
            /\ UNCHANGED <<epoch, rev, content, writes, messages, stopped,
                            targetEpoch, targetRev, completed, badClaim, badAck>>
Observe == /\ targetEpoch # 0 /\ ~completed /\ localRev >= targetRev
           /\ (Policy \in {"Current", "GapOnly"} \/ localEpoch = targetEpoch)
           /\ completed' = TRUE
           /\ badAck' = (localEpoch # targetEpoch \/ badClaim)
           /\ UNCHANGED <<epoch, rev, content, writes, messages, localEpoch,
                           localRev, localContent, refresh, stopped,
                           targetEpoch, targetRev, badClaim>>
Quiesce == /\ ~stopped /\ stopped' = TRUE
           /\ UNCHANGED <<epoch, rev, content, writes, messages, localEpoch,
                           localRev, localContent, refresh, targetEpoch,
                           targetRev, completed, badClaim, badAck>>
Next == Write \/ Expire \/ Drop \/ Deliver \/ Snapshot \/ Observe \/ Quiesce
Spec == Init /\ [][Next]_vars /\ WF_vars(Snapshot) /\ WF_vars(Drop)
NoIncompleteClaims == ~badClaim
WaiterSound == ~badAck
Converges == stopped ~> (<<localEpoch,localRev,localContent>> = <<epoch,rev,content>>)
=============================================================================
