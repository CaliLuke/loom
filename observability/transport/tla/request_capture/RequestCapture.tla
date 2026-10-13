------------------------- MODULE RequestCapture -------------------------
EXTENDS Naturals, Sequences, TLC
CONSTANT Mode
VARIABLES started, done, reads, sourceReads, received, original, closes,
          sourceCloses, retained, omitted, eof, steps
vars == <<started, done, reads, sourceReads, received, original, closes,
          sourceCloses, retained, omitted, eof, steps>>
Init == /\ started = FALSE /\ done = FALSE /\ reads = 0 /\ sourceReads = 0
        /\ received = <<>> /\ original = <<>> /\ closes = 0 /\ sourceCloses = 0
        /\ retained = 0 /\ omitted = FALSE /\ eof = FALSE /\ steps = 0
Start == /\ ~started /\ started' = TRUE
         /\ sourceReads' = (IF Mode = "eager" THEN 1 ELSE sourceReads)
         /\ UNCHANGED <<done, reads, received, original, closes, sourceCloses,
                        retained, omitted, eof, steps>>
Read(n, err) == /\ started /\ ~done /\ steps < 3
 /\ reads' = reads + 1 /\ sourceReads' = sourceReads + 1
 /\ original' = Append(original, <<n, err>>)
 /\ received' = Append(received, IF Mode = "replace" /\ err = "boom"
                                  THEN <<1, "none">> ELSE <<n, err>>)
 /\ omitted' = (omitted \/ (Mode # "unbounded" /\ retained+n > 2))
 /\ retained' = (IF omitted' THEN 0 ELSE retained+n)
 /\ eof' = (eof \/ err = "eof")
 /\ steps' = steps + 1
 /\ UNCHANGED <<started, done, closes, sourceCloses>>
Close == /\ started /\ ~done /\ steps < 3
 /\ closes' = closes + 1
 /\ sourceCloses' = sourceCloses + (IF Mode = "lost-close" THEN 0 ELSE 1)
 /\ steps' = steps + 1
 /\ UNCHANGED <<started, done, reads, sourceReads, received, original,
                retained, omitted, eof>>
Return == /\ started /\ ~done /\ done' = TRUE
 /\ UNCHANGED <<started, reads, sourceReads, received, original, closes,
                sourceCloses, retained, omitted, eof, steps>>
Next == Start \/ (\E n \in 0..2, err \in {"none", "boom", "eof"}: Read(n, err))
        \/ Close \/ Return \/ (done /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
NoExtraRead == sourceReads = reads
ReadIdentity == received = original
CloseOwnership == closes = sourceCloses
BoundedPreview == retained <= 2 /\ (omitted => retained = 0)
=============================================================================
