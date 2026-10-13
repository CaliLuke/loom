---------------------- MODULE ErrorConstructorIdentity ----------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
Errors == {"foo_bar", "FooBar", "foo_bar2"}
Types == {"MakeFooBar", "MakeFooBar2"}
Base(error) == IF error = "foo_bar2" THEN "MakeFooBar2" ELSE "MakeFooBar"
References == <<"foo_bar", "FooBar", "foo_bar", "foo_bar2">>
Candidates(name) == <<name, name \o "2", name \o "3", name \o "4", name \o "5", name \o "6", name \o "7", name \o "8", name \o "9", name \o "10">>
Fresh(name, used) ==
    LET choices == Candidates(name)
        free == {i \in 1..Len(choices) : choices[i] \notin used}
        first == CHOOSE i \in free : \A j \in free : i <= j
    IN choices[first]

VARIABLES pendingErrors, pendingTypes, occupied, factoryNames, typeNames, refs
vars == <<pendingErrors, pendingTypes, occupied, factoryNames, typeNames, refs>>
Init == /\ pendingErrors = Errors
        /\ pendingTypes = Types
        /\ occupied = {"Client"}
        /\ factoryNames = [e \in Errors |-> ""]
        /\ typeNames = [t \in Types |-> ""]
        /\ refs = <<>>
Factory(error) ==
    LET prior == {factoryNames[e] : e \in Errors \ pendingErrors}
        name == CASE Mode = "Concatenated" -> Base(error)
                  [] Mode = "Separate" -> Fresh(Base(error), prior)
                  [] OTHER -> Fresh(Base(error), occupied)
    IN /\ error \in pendingErrors
       /\ pendingErrors' = pendingErrors \ {error}
       /\ factoryNames' = [factoryNames EXCEPT ![error] = name]
       /\ occupied' = IF Mode \in {"Concatenated", "Separate"}
                       THEN occupied ELSE occupied \cup {name}
       /\ UNCHANGED <<pendingTypes, typeNames, refs>>
Nominal(type) ==
    LET name == Fresh(type, occupied)
    IN /\ pendingErrors = {}
       /\ type \in pendingTypes
       /\ pendingTypes' = pendingTypes \ {type}
       /\ typeNames' = [typeNames EXCEPT ![type] = name]
       /\ occupied' = occupied \cup {name}
       /\ UNCHANGED <<pendingErrors, factoryNames, refs>>
Reference ==
    /\ pendingErrors = {} /\ pendingTypes = {}
    /\ Len(refs) < Len(References)
    /\ LET error == References[Len(refs) + 1]
           name == IF Mode = "Uncached" THEN Fresh(Base(error), occupied)
                   ELSE factoryNames[error]
       IN /\ refs' = Append(refs, name)
          /\ occupied' = IF Mode = "Uncached" THEN occupied \cup {name}
                          ELSE occupied
    /\ UNCHANGED <<pendingErrors, pendingTypes, factoryNames, typeNames>>
Next == (\E error \in pendingErrors : Factory(error))
        \/ (\E type \in pendingTypes : Nominal(type))
        \/ Reference
        \/ (Len(refs) = Len(References) /\ UNCHANGED vars)
DistinctFactories == \A a,b \in Errors \ pendingErrors :
    a # b => factoryNames[a] # factoryNames[b]
PackageDeclarationsDistinct == \A e \in Errors \ pendingErrors :
    \A t \in Types \ pendingTypes : factoryNames[e] # typeNames[t]
ReferencesAgree == \A i \in 1..Len(refs) : refs[i] = factoryNames[References[i]]
=============================================================================
