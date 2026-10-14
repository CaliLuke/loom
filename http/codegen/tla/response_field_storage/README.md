# Selected response constructor storage

This bounded model explores the assignment boundary after a response body has
been transformed into `v` and before it is inserted into the selected result
field. It covers ordinary service and projected-view destinations, required and
optional scalars, defaults, named scalars, nullable wrappers, arrays and bytes.
The eleven declaration facts are inputs checked by the Go readiness table; the
model does not reimplement or prove the field-layout classifier.

The reproduced `service` design uses only service requiredness/defaults to select
`&v`. It fails on a required viewed scalar: the view declares a pointer but the
constructor inserts a value. Optional nullable fields show the inverse mismatch.
The `context` alternative uses AttributeContext.IsPrimitivePointer directly;
it fails on a nullable viewed scalar and would also address arrays. Context's
blanket Pointer flag is not the declaration's physical storage decision.
The `layout` design consults the selected destination's canonical field layout
and adds an address only for the primitive transformation boundary. It satisfies
AssignmentMatchesDeclaration for all eleven cases.

Implementation seams:

- `http/codegen/service_data_response.go`: buildResponseResultInit selects the
  result or view return type; buildResponseResultInitCode has that same target
  context, and buildResponseResultPointer now retains it through AttributeContext.IsPrimitivePointer.
- `expr.ServiceFieldLayout`: existing owner of pointer and explicit-presence
  storage, consumed by `codegen.NameScope.objectFieldTypeDef` for declarations.
- `http/codegen/type_sections.go`: appendInitResult consumes the address flag;
  object/union transforms already have their own reference semantics. The repair
  must not apply layout.Pointer blindly to those nonprimitive transformations.
- `http/codegen/selected_response_storage_readiness_test.go`: correspondence
  cases and actual transform snippets. Both HTTP and JSON-RPC use the HTTP
  response-analysis owner; generated module compilation checks its emitted types.

Assumptions and limits: selected field identity and target context are already
correct; the canonical declaration classifier is trusted as input. The scalar
transform yields its target value (including a nullable wrapper), before the
outer declaration pointer is considered. The model excludes whole bodies,
empty/object selections, unions' own address/dereference paths, arbitrary custom
codecs, validation and default application. Those are unchanged boundaries and
need bounded generated controls, not claims from this proof. This model says
nothing about JSON-RPC success-envelope projection (#853).

Run with TLA+ tools 1.7.4, JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
export TLA_JAR=/absolute/path/to/tla2tools-1.7.4.jar
cd http/codegen/tla/response_field_storage
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config service.cfg ResponseFieldStorage
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config context.cfg ResponseFieldStorage
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config layout.cfg ResponseFieldStorage
```

Corretto 25.0.4.1 checks at parent fce66acb137cf7df0282bfe69446d9cceca1ceee:
service exits 12 with the required-view counterexample (12 states); context exits
12 with the nullable-view counterexample (18 states); layout exits 0 after 22
states. Both negative runs fail the assignment invariant, not parsing. Use
separate temporary Java/checker directories for runs; do not commit traces.


## Selected source conversion

The generated named-view fixture exposes the source side of the same boundary:
the original buildResponseBodyInit detached a field with serviceBodyTransformSource
and called marshal as though the selected root were a value. The request constructor has a
separate semantic-pointer workaround. Both must derive the occurrence layout
before detaching it. Native pointer response conversion must preserve absence,
not merely insert a dereference. Constructor body references and return values
must describe the same storage. #847's selected field identity is independent.

`SelectedSourceStorage` checks required/optional named views, optional/required
named service fields and opaque nullable wrappers. Pointer declaration facts are
inputs. Only optional native pointer fields may be absent; required-view admission
is owned by existing generated service projection/validation. Present conversion
must read exactly the declared pointer depth, absent values must never be read,
and absent source pointers must produce absent body pointers. Wrapper contents,
codec behavior and generated client null admission are outside this model.

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config source-current.cfg SelectedSourceStorage
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config source-dereference.cfg SelectedSourceStorage
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config source-presence.cfg SelectedSourceStorage
```

Current exits12 at SourceReadMatchesStorage (8 states); dereference exits12 at
NoAbsentRead (9 states); presence exits0 after14 states. This rejects a tempting
compile-only fix that would panic on nil optional named fields. The original
target model remains unchanged and its prior results remain valid.


Implemented correspondence: AttributeContext.IsPrimitivePointer queries
expr.ServiceFieldLayout for the enclosing native primitive occurrence.
Target reconstruction consumes that flag. Request extraction retains its prior
external admission guard. Response construction passes the flag to
marshalResponseBodySource, which declares a pointer result, guards source nil,
converts the dereferenced present value and retains its address. The same return
reference populates TypeData.Ref. No public requiredness or nullable admission
rule changes. `codegen/generator/selected_response_storage_test.go` exercises the
real service/transport generator entry points, both client constructors and
mounted HTTP with required/optional named, nullable null/value, native zeros,
custom time.Time, array/bytes, ordinary service and whole-body controls.
The twelve target seam cases include a supported custom scalar beyond the
original eleven modeled inputs. A separate context-flags seam checks defaults. Required/optional named source
controls include both projected views and ordinary service declarations.
Constructor allocation for two distinct source views sharing one named body
alias remains #735; its captured failing aggregate is not claimed fixed here.

#887 removes the selected-body-only primitive classifier and uses that shared
context query for both selected bodies and transport-bound headers. The query
now locates the actual field and excludes nonprimitive carriers before consulting
ServiceFieldLayout; its Pointer flag alone is insufficient. FieldPresence still
owns transport wrappers. Direct carrier/binding tests and the original named-slice
view header compile/wire witness check the previously uncovered consumer.


## Constructor declaration context (#930)

The storage models above assume the selected target context is correct. The
`ConstructorContext` model explores that premise separately: return declaration
and mapped-field casts must use the same package and allocated nominal identity.
The finite declaration facts cover ordinary service, projected view, a view whose
projection scope allocates a suffixed name, and a located shared-package alias.
They are checked against direct Go analysis and actual generated compilation;
the model does not prove NameScope allocation or import resolution.

`split` reproduces the old response constructor: selected return declaration,
service package/scope for fields. `package` corrects only the default package and
still fails on a nominal collision. `selected` derives both from the already
selected AttributeContext. Body transformation and physical pointer layout stay
with their existing owners; this change concerns declaration identity only.

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config constructor-split.cfg ConstructorContext
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config constructor-package.cfg ConstructorContext
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config constructor-selected.cfg ConstructorContext
```

TLC1.7.4 on Corretto25 reports split6 states and package7 states, both exit12
on AssignmentNamesDeclaration; selected exits0 after8 states. The nominal Go
witness authors `Label` and `label`: the projected declarations allocate
`LabelView` and `LabelView2`, while service-scope casting loses the suffix even
when the default package is manually corrected.
The selected response context must reach `fieldCode` even though other constructor
owners still supply their own existing context. No global InitData redesign,
codec handling, defaults or view-selection policy is implied.


Implemented correspondence: `buildResponsesFromIR` selects the AttributeContext;
`buildResponseResultInit` derives return name/ref/package and naming scope from
it, retaining it as private InitData.returnContext. `fieldCode` consumes that
context through finalization and direct rendering, while other constructor
owners retain their existing supplied scope. `TestProjectedConstructorTargetContext`
covers header/cookie, service/view and required/optional combinations;
`TestProjectedConstructorNominalScope` is the real allocated-name witness.
`TestProjectedConstructorTargetCompiles` compiles those eight generated shapes
and exercises empty/nonempty supplied values and optional absence, plus the
collision shape. Existing located-package/import-alias and selected-body
HTTP/JSON-RPC controls remain green. This does not establish arbitrary custom
codec/default behavior or change invalid-design admission.
