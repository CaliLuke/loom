---------------------------- MODULE PerDoOther ----------------------------
EXTENDS PerDo
\* Targeted current counterexample partition; the underlying transitions and
\* independent per-exchange history oracle are unchanged.
OnlyNon100 == \A i \in 1..Len(s.history): s.history[i].code # 100
=============================================================================
