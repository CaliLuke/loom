# Union code generation in plugins

Plugins that generate typed JSON input should use `codegen/uniongen` for Go sum
union declarations and codecs, and `codegen/unionjson` for untagged branch
analysis. These are the same owners used by Loom service and HTTP generation.
Do not implement a second discriminator, null-admission or untagged matcher.

Build a `uniongen.Type` from the finalized `expr.Union`. Allocate names and Go
references with `codegen.NameScope`; populate each `uniongen.Field` with the
branch's authored tag and `expr.AllowsNull` result. The renderer does not choose
physical field storage: references and conversion contexts must describe the
same declared types. Set `JSONOnly: true` when form codecs are not needed;
the zero value retains JSON and form methods.

For untagged unions, build one `unionjson.Branch` per declared variant in order,
including its allocated type, field, kind constant and validation code. Call
`unionjson.Analyze` with the finalized union, the generator's member-name policy
and object-closure policy, and assign the result to `uniongen.Type.JSON`.
Set `Untagged` from the expression. Analysis rejects invalid expression plans;
plugins must pass evaluated, valid designs. The generated matcher preserves the
unique match requirement in both the schema and decoded Go representation.

Call `uniongen.Emit` inside a `codegen.JenniferSection`. Generated files need
`encoding/json/v2` as `json`, `encoding/json/jsontext` as `jsontext`, `fmt`,
`github.com/CaliLuke/loom/pkg` as `loom`, and imports required by branch types
and validators. Form codecs additionally need `net/url` and
`github.com/CaliLuke/loom/http` as `loomhttp`. Loom's file rendering removes
unused imports. Generation of user types, validation helpers, conversion plans,
and transport dispatch remains the plugin's responsibility.

The built-in generators and wire contracts are unchanged by this extension API.
