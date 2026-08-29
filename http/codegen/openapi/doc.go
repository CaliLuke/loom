/*
Package openapi provides the active schema model and shared helpers used to
generate OpenAPI 3.1 and 3.2 specifications from Loom designs.

The exported schema, tag, and external-documentation types are part of the v3
document model. Rendering starts with the v3 package and keeps schema analysis
state per invocation in internal/ir. This package intentionally does not
provide the legacy JSON Hyper-Schema generator, global definitions registry,
or compatibility aliases for those removed entry points.
*/
package openapi
