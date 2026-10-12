-------------------------- MODULE ReaderCursor --------------------------
EXTENDS Naturals, FiniteSets, TLC
CONSTANT Mode
Streams == {"a", "b"}
VARIABLES phase, last, cursor, owed, delivered, closing, publishedAfterClose
vars == <<phase, last, cursor, owed, delivered, closing, publishedAfterClose>>
Init == /\ phase = [s \in Streams |-> "unprepared"]
        /\ last = [s \in Streams |-> 0]
        /\ cursor = [s \in Streams |-> 0]
        /\ owed = [s \in Streams |-> {}]
        /\ delivered = [s \in Streams |-> {}]
        /\ closing = FALSE
        /\ publishedAfterClose = FALSE
Prepare(s) == /\ phase[s] = "unprepared" /\ ~closing
              /\ phase' = [phase EXCEPT ![s] = "prepared"]
              /\ cursor' = [cursor EXCEPT ![s] = last[s]]
              /\ UNCHANGED <<last, owed, delivered, closing, publishedAfterClose>>
Fail(s) == /\ phase[s] = "prepared"
           /\ phase' = [phase EXCEPT ![s] = "failed"]
           /\ UNCHANGED <<last, cursor, owed, delivered, closing, publishedAfterClose>>
Publish(s) == /\ phase[s] = "prepared"
              /\ (Mode = "unchecked-publication" \/ ~closing)
              /\ phase' = [phase EXCEPT ![s] = "published"]
              /\ publishedAfterClose' = (publishedAfterClose \/ closing)
              /\ UNCHANGED <<last, cursor, owed, delivered, closing>>
Produce(s) == /\ last[s] < 2
              /\ last' = [last EXCEPT ![s] = @ + 1]
              /\ owed' = IF phase[s] \in {"prepared", "published"}
                           THEN [owed EXCEPT ![s] = @ \cup {last[s] + 1}]
                           ELSE owed
              /\ UNCHANGED <<phase, cursor, delivered, closing, publishedAfterClose>>
Poll(s) == /\ phase[s] = "published" /\ ~closing
           /\ LET start == IF Mode = "legacy" /\ delivered[s] = {}
                           THEN last[s] ELSE cursor[s]
              IN delivered' = [delivered EXCEPT ![s] = @ \cup ((start + 1)..last[s])]
           /\ cursor' = [cursor EXCEPT ![s] = last[s]]
           /\ UNCHANGED <<phase, last, owed, closing, publishedAfterClose>>
Close == /\ ~closing /\ closing' = TRUE
         /\ UNCHANGED <<phase, last, cursor, owed, delivered, publishedAfterClose>>
Next == Close \/ \E s \in Streams: Prepare(s) \/ Fail(s) \/ Publish(s) \/ Produce(s) \/ Poll(s)
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in [Streams -> {"unprepared", "prepared", "published", "failed"}]
          /\ last \in [Streams -> 0..2] /\ cursor \in [Streams -> 0..2]
          /\ owed \in [Streams -> SUBSET (1..2)]
          /\ delivered \in [Streams -> SUBSET (1..2)]
          /\ closing \in BOOLEAN /\ publishedAfterClose \in BOOLEAN
NoLostEvents == \A s \in Streams: phase[s] = "published" =>
                 \A id \in owed[s]: id \in delivered[s] \/ id > cursor[s]
NoPostClosePublication == ~publishedAfterClose
=============================================================================
