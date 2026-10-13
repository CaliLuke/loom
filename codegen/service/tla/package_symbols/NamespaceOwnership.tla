------------------------ MODULE NamespaceOwnership -------------------------
EXTENDS Naturals, Sequences, FiniteSets
CONSTANT Mode
Fixed == {"Client"}
Types == {"Foo", "FooServerStream", "FooServerStream2"}
Methods == {"Client", "Foo"}
Candidates(name) == <<name, name \o "2", name \o "3", name \o "4", name \o "5">>
FirstFree(name, used) ==
    LET choices == Candidates(name)
        free == {i \in 1..Len(choices) : choices[i] \notin used}
        first == CHOOSE i \in free : \A j \in free : i <= j
    IN choices[first]
VARIABLES phase, pendingTypes, pendingMethods, packageNames, memberNames,
          typeNames, methodNames, helper
vars == <<phase, pendingTypes, pendingMethods, packageNames, memberNames,
          typeNames, methodNames, helper>>
Init == /\ phase = "methods"
        /\ pendingTypes = Types
        /\ pendingMethods = Methods
        /\ packageNames = Fixed
        /\ memberNames = {}
        /\ typeNames = [name \in Types |-> ""]
        /\ methodNames = [name \in Methods |-> ""]
        /\ helper = ""
AllocateMethod(name) ==
    LET used == IF Mode = "Shared" THEN packageNames ELSE memberNames
        selected == FirstFree(name, used)
    IN /\ phase = "methods"
       /\ name \in pendingMethods
       /\ pendingMethods' = pendingMethods \ {name}
       /\ methodNames' = [methodNames EXCEPT ![name] = selected]
       /\ memberNames' = memberNames \cup {selected}
       /\ packageNames' = IF Mode = "Shared" THEN packageNames \cup {selected}
                          ELSE packageNames
       /\ UNCHANGED <<phase, pendingTypes, typeNames, helper>>
PrepareHelpers ==
    /\ phase = "methods"
    /\ pendingMethods = {}
    /\ helper' = methodNames["Foo"] \o "ServerStream"
    /\ packageNames' = IF Mode = "Owned" THEN packageNames \cup {helper'}
                       ELSE packageNames
    /\ phase' = "types"
    /\ UNCHANGED <<pendingTypes, pendingMethods, memberNames, typeNames, methodNames>>
AllocateType(name) ==
    LET selected == FirstFree(name, packageNames)
    IN /\ phase = "types"
       /\ name \in pendingTypes
       /\ pendingTypes' = pendingTypes \ {name}
       /\ packageNames' = packageNames \cup {selected}
       /\ typeNames' = [typeNames EXCEPT ![name] = selected]
       /\ UNCHANGED <<phase, pendingMethods, memberNames, methodNames, helper>>
Next == (\E name \in pendingMethods : AllocateMethod(name))
        \/ PrepareHelpers
        \/ (\E name \in pendingTypes : AllocateType(name))
        \/ (phase = "types" /\ pendingTypes = {} /\ UNCHANGED vars)
IndependentMethods == \A name \in Methods \ pendingMethods : methodNames[name] = name
PackageDeclarationsDistinct ==
    /\ \A name \in Types \ pendingTypes : typeNames[name] \notin Fixed \cup {helper}
    /\ \A a,b \in Types \ pendingTypes : a # b => typeNames[a] # typeNames[b]
=============================================================================
