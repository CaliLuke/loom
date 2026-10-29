-------------------------- MODULE SchemaExamples --------------------------
EXTENDS Naturals, Sequences
CONSTANT Mode
VARIABLES source, media, output, mediaOutput, visits
vars == <<source, media, output, mediaOutput, visits>>
Inputs == {<<"red", "blue">>, <<"null", "blue", "blue">>, <<"red">>}
Init == /\ source \in Inputs /\ media = <<"green">>
        /\ output = <<>> /\ mediaOutput = <<>> /\ visits = 0
Emit == /\ visits < 2
        /\ output' = CASE Mode = "representative" -> <<source[Len(source)]>>
                         [] Mode = "collapsed" -> source \o media
                         [] Mode = "append" -> output \o source
                         [] OTHER -> source
        /\ mediaOutput' = IF Mode = "collapsed" THEN source \o media ELSE media
        /\ visits' = visits + 1
        /\ UNCHANGED <<source, media>>
Spec == Init /\ [][Emit]_vars
OrderedValuesPreserved == visits > 0 => output = source /\ mediaOutput = media
=============================================================================
