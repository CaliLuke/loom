---------------------------- MODULE EmptyPoll ----------------------------
EXTENDS Naturals
CONSTANTS ReserveResponse, CallerBudget
VARIABLES elapsed, result
vars == <<elapsed, result>>
Poll == 5
OrdinaryBudget == 5
(* A healthy server finishes its poll, then needs one scheduling/network step.
   This models a response faster than the ordinary I/O budget, not arbitrary
   transport delay. CallerBudget=0 means no stricter authored deadline. *)
ResponseAt == Poll + 1
OwnedBudget == IF ReserveResponse THEN Poll + OrdinaryBudget ELSE OrdinaryBudget
Deadline == IF CallerBudget > 0 /\ CallerBudget < OwnedBudget
            THEN CallerBudget ELSE OwnedBudget
Init == /\ elapsed = 0 /\ result = "pending"
Tick == /\ result = "pending" /\ elapsed < ResponseAt /\ elapsed < Deadline
        /\ elapsed' = elapsed + 1 /\ UNCHANGED result
Reply == /\ result = "pending" /\ elapsed = ResponseAt
         /\ result' = "redis.Nil" /\ UNCHANGED elapsed
Expire == /\ result = "pending" /\ elapsed = Deadline
          /\ result' = "timeout" /\ UNCHANGED elapsed
Next == Tick \/ Reply \/ Expire
Spec == Init /\ [][Next]_vars /\ WF_vars(Tick) /\ WF_vars(Reply) /\ WF_vars(Expire)
TypeOK == /\ elapsed \in 0..ResponseAt
          /\ result \in {"pending", "redis.Nil", "timeout"}
HealthyReply == CallerBudget = 0 => result # "timeout"
CallerLimit == CallerBudget > 0 /\ CallerBudget < ResponseAt => result # "redis.Nil"
Completes == <> (result # "pending")
=============================================================================
