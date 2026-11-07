---------------- MODULE ResponsePreparationFailure ----------------
EXTENDS Naturals
CONSTANT Repair
VARIABLES phase, channel, hasID, sourceOK, conversionOK, codecOK,
          attempts, envelope, returned, diagnostic
vars == <<phase, channel, hasID, sourceOK, conversionOK, codecOK,
          attempts, envelope, returned, diagnostic>>
Init == /\ phase = "choose"
        /\ channel = "unary" /\ hasID = TRUE
        /\ sourceOK = TRUE /\ conversionOK = TRUE /\ codecOK = TRUE
        /\ attempts = 0 /\ envelope = "none"
        /\ returned = FALSE /\ diagnostic = FALSE
Choose == /\ phase = "choose"
          /\ \E c \in {"unary", "stream"}, i,s,v,k \in BOOLEAN:
               /\ channel' = c /\ hasID' = i
               /\ sourceOK' = s /\ conversionOK' = v /\ codecOK' = k
          /\ phase' = "prepare"
          /\ UNCHANGED <<attempts,envelope,returned,diagnostic>>
Prepare == /\ phase = "prepare"
           /\ IF channel = "unary" /\ ~hasID THEN
                /\ phase' = "done"
                /\ UNCHANGED <<attempts,envelope,returned,diagnostic>>
              ELSE IF sourceOK /\ conversionOK THEN
                /\ phase' = "encode" /\ envelope' = "success"
                /\ UNCHANGED <<attempts,returned,diagnostic>>
              ELSE IF channel = "stream" THEN
                /\ phase' = "done" /\ returned' = TRUE
                /\ UNCHANGED <<attempts,envelope,diagnostic>>
              ELSE IF Repair = "current" THEN
                /\ phase' = "done" /\ diagnostic' = TRUE
                /\ UNCHANGED <<attempts,envelope,returned>>
              ELSE
                /\ phase' = "encode" /\ envelope' = "error"
                /\ UNCHANGED <<attempts,returned,diagnostic>>
           /\ UNCHANGED <<channel,hasID,sourceOK,conversionOK,codecOK>>
Encode == /\ phase = "encode"
          /\ attempts' = IF ~codecOK /\ Repair = "retry-write" THEN 2 ELSE 1
          /\ diagnostic' = (~codecOK /\ channel = "unary")
          /\ returned' = (~codecOK /\ channel = "stream")
          /\ phase' = "done"
          /\ UNCHANGED <<channel,hasID,sourceOK,conversionOK,codecOK,envelope>>
Next == Choose \/ Prepare \/ Encode
FailureFramed == phase # "done" \/ channel # "unary" \/ ~hasID \/
                 (sourceOK /\ conversionOK) \/ (envelope = "error" /\ attempts = 1)
NotificationSuppressed == phase # "done" \/ channel # "unary" \/ hasID \/ attempts = 0
OnePublicationAttempt == attempts <= 1
StreamFailureReturned == phase # "done" \/ channel # "stream" \/
                         (sourceOK /\ conversionOK) \/ (returned /\ attempts = 0)
ValidBodyRetained == phase # "done" \/ ~(sourceOK /\ conversionOK) \/
                    (channel = "unary" /\ ~hasID) \/ envelope = "success"
ReachError == ~(phase = "done" /\ envelope = "error" /\ attempts = 1 /\ ~codecOK)
ReachStream == ~(phase = "done" /\ channel = "stream" /\ returned /\ attempts = 0)
=============================================================================
