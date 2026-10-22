-------------------------- MODULE RenderScheduler --------------------------
EXTENDS Naturals, FiniteSets
CONSTANT FirstCompletion
VARIABLES failures, next, active, completed, stopped, cutoff, reported, returned, retained
vars == <<failures, next, active, completed, stopped, cutoff, reported, returned, retained>>
Jobs == 1..4
Workers == 2
Minimum(values) == CHOOSE n \in values : \A other \in values : n <= other
Init == /\ failures \in SUBSET Jobs
        /\ next = 1 /\ active = {} /\ completed = {}
        /\ stopped = FALSE /\ cutoff = 0 /\ reported = 0 /\ returned = FALSE /\ retained = Jobs
Dispatch ==
 /\ ~stopped /\ ~returned /\ next \in Jobs /\ Cardinality(active) < Workers
 /\ active' = active \cup {next} /\ next' = next + 1
 /\ UNCHANGED <<failures, completed, stopped, cutoff, reported, returned, retained>>
Complete(job) ==
 /\ job \in active
 /\ active' = active \ {job} /\ completed' = completed \cup {job}
 /\ retained' = IF stopped \/ job \in failures THEN active \ {job} ELSE retained \ {job}
 /\ stopped' = (stopped \/ job \in failures)
 /\ cutoff' = IF ~stopped /\ job \in failures THEN next ELSE cutoff
 /\ reported' = IF job \notin failures THEN reported
                 ELSE IF reported = 0 THEN job
                 ELSE IF FirstCompletion THEN reported ELSE Minimum({reported,job})
 /\ UNCHANGED <<failures, next, returned>>
Return ==
 /\ ~returned /\ active = {} /\ (stopped \/ next = 5)
 /\ returned' = TRUE
 /\ UNCHANGED <<failures, next, active, completed, stopped, cutoff, reported, retained>>
Next == Dispatch \/ (\E job \in Jobs : Complete(job)) \/ Return
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Bounded == Cardinality(active) <= Workers
UniqueDispatch == active \cap completed = {} /\ active \cup completed = 1..(next-1)
StoppedDispatch == stopped => next = cutoff
Joined == returned => active = {}
CorrectError == returned /\ failures # {} => reported = Minimum(failures)
SuccessComplete == returned /\ reported = 0 => completed = Jobs /\ failures = {}
NoEarlyRelease == active \subseteq retained
ReleasedOnReturn == returned => retained = {}
Completes == <>returned
=============================================================================
