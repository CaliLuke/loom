----------------------- MODULE SinkGroupRecovery -----------------------
EXTENDS Naturals
CONSTANT Strategy
VARIABLES active, closing, group, member, consumer, phase, captured,
          creationStart, failures, destroyPhase, removed, readded,
          transientUsed, published, badRevival, badStart, badPublication, pending
vars == <<active, closing, group, member, consumer, phase, captured,
          creationStart, failures, destroyPhase, removed, readded,
          transientUsed, published, badRevival, badStart, badPublication, pending>>
Start(e) == IF e = 1 THEN 7 ELSE 11
Locked == Strategy # "snapshot" /\ phase \in {"repair", "published", "created"}
Init == /\ active = 1 /\ closing = FALSE /\ group = TRUE
        /\ member = 1 /\ consumer = TRUE /\ phase = "idle" /\ captured = 0
        /\ creationStart = 7 /\ failures = 0 /\ destroyPhase = 0
        /\ removed = FALSE /\ readded = FALSE /\ transientUsed = FALSE
        /\ published = FALSE /\ badRevival = FALSE /\ badStart = FALSE
        /\ badPublication = FALSE /\ pending = FALSE
DestroyGroup == /\ destroyPhase = 0 /\ group' = FALSE /\ consumer' = FALSE
                /\ destroyPhase' = 1
                /\ UNCHANGED <<active, closing, member, phase, captured, creationStart,
                     failures, removed, readded, transientUsed, published,
                     badRevival, badStart, badPublication, pending>>
DestroyMembership == /\ destroyPhase = 1 /\ member' = 0 /\ destroyPhase' = 2
                     /\ UNCHANGED <<active, closing, group, consumer, phase, captured,
                          creationStart, failures, removed, readded, transientUsed,
                          published, badRevival, badStart, badPublication, pending>>
\* Both a read and replacement-consumer preparation can discover NOGROUP.
Missing(entry) == /\ entry \in {"read", "rotation"}
                 /\ phase = "idle" /\ active # 0 /\ ~closing /\ ~group
                 /\ failures < 2 /\ failures' = failures + 1
                 /\ phase' = (IF Strategy = "current" THEN "idle" ELSE "missing")
                 /\ pending' = TRUE
                 /\ UNCHANGED <<active, closing, group, member, consumer, captured,
                      creationStart, destroyPhase, removed, readded, transientUsed,
                      published, badRevival, badStart, badPublication>>
\* Existing replica state reveals a membership reset even if the group survived.
ObserveReset == /\ phase = "idle" /\ active # 0 /\ ~closing /\ (member # active \/ pending)
                /\ Strategy # "current" /\ phase' = "missing"
                /\ pending' = TRUE
                /\ UNCHANGED <<active, closing, group, member, consumer, captured,
                     creationStart, failures, destroyPhase, removed, readded,
                     transientUsed, published, badRevival, badStart, badPublication>>
BeginRepair == /\ phase = "missing" /\ active # 0 /\ ~closing
               /\ phase' = "repair" /\ captured' = active /\ published' = FALSE
               /\ UNCHANGED <<active, closing, group, member, consumer, creationStart,
                    failures, destroyPhase, removed, readded, transientUsed,
                    badRevival, badStart, badPublication, pending>>
AbortRemoved == /\ phase = "missing" /\ (active = 0 \/ closing)
                /\ phase' = "cooldown"
                /\ UNCHANGED <<active, closing, group, member, consumer, captured,
                     creationStart, failures, destroyPhase, removed, readded,
                     transientUsed, published, badRevival, badStart, badPublication, pending>>
PublishMember == /\ phase = "repair" /\ member' = captured /\ published' = TRUE
                 /\ phase' = "published"
                 /\ UNCHANGED <<active, closing, group, consumer, captured,
                      creationStart, failures, destroyPhase, removed, readded,
                      transientUsed, badRevival, badStart, badPublication, pending>>
CreateGroup == /\ phase = "published" /\ phase' = "created" /\ group' = TRUE
               /\ creationStart' = (IF group THEN creationStart
                    ELSE IF Strategy = "main-start" THEN 0 ELSE Start(captured))
               /\ badRevival' = (badRevival \/ active # captured \/ closing)
               /\ badStart' = (badStart \/ (~group /\ Strategy = "main-start"))
               /\ badPublication' = (badPublication \/ ~published)
               /\ UNCHANGED <<active, closing, member, consumer, captured, failures,
                    destroyPhase, removed, readded, transientUsed, published, pending>>
CreateConsumer == /\ phase = "created" /\ consumer' = group
                  /\ phase' = "cooldown" /\ pending' = ~group
                  /\ UNCHANGED <<active, closing, group, member, captured, creationStart,
                       failures, destroyPhase, removed, readded, transientUsed,
                       published, badRevival, badStart, badPublication>>
TransientFailure == /\ phase \in {"repair", "published", "created"} /\ ~transientUsed
                    /\ transientUsed' = TRUE /\ phase' = "cooldown"
                    /\ UNCHANGED <<active, closing, group, member, consumer, captured,
                         creationStart, failures, destroyPhase, removed, readded,
                         published, badRevival, badStart, badPublication, pending>>
Wait == /\ phase = "cooldown" /\ phase' = "idle" /\ failures' = 0
        /\ UNCHANGED <<active, closing, group, member, consumer, captured, creationStart,
             destroyPhase, removed, readded, transientUsed, published,
             badRevival, badStart, badPublication, pending>>
Remove == /\ ~Locked /\ ~removed /\ ~closing /\ active # 0
          /\ active' = 0 /\ member' = 0 /\ group' = FALSE /\ consumer' = FALSE
          /\ removed' = TRUE
          /\ UNCHANGED <<closing, phase, captured, creationStart, failures,
               destroyPhase, readded, transientUsed, published,
               badRevival, badStart, badPublication, pending>>
Readd == /\ ~Locked /\ removed /\ ~readded /\ ~closing /\ active = 0
         /\ active' = 2 /\ member' = 2 /\ group' = TRUE /\ consumer' = TRUE
         /\ readded' = TRUE /\ creationStart' = Start(2)
         /\ UNCHANGED <<closing, phase, captured, failures, destroyPhase, removed,
              transientUsed, published, badRevival, badStart, badPublication, pending>>
Close == /\ ~Locked /\ ~closing /\ closing' = TRUE
         /\ UNCHANGED <<active, group, member, consumer, phase, captured,
              creationStart, failures, destroyPhase, removed, readded, transientUsed,
              published, badRevival, badStart, badPublication, pending>>
Next == DestroyGroup \/ DestroyMembership \/ (\E e \in {"read", "rotation"}: Missing(e))
        \/ ObserveReset \/ BeginRepair \/ AbortRemoved \/ PublishMember \/ CreateGroup
        \/ CreateConsumer \/ TransientFailure \/ Wait \/ Remove \/ Readd \/ Close
Spec == Init /\ [][Next]_vars
        /\ WF_vars(Missing("read")) /\ WF_vars(ObserveReset)
        /\ WF_vars(BeginRepair) /\ WF_vars(AbortRemoved)
        /\ WF_vars(PublishMember) /\ WF_vars(CreateGroup)
        /\ WF_vars(CreateConsumer) /\ WF_vars(Wait)
NoHotRetry == failures <= 1
NoRemovedRevival == ~badRevival
OwnStart == ~badStart
MembershipBeforeCreation == ~badPublication
Recovery == <>[](closing \/ active = 0 \/ (group /\ member = active /\ consumer))
=============================================================================
