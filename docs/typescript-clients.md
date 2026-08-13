---
title: TypeScript Clients
weight: 5
description: "Generate a typed TypeScript API client from Loom's OpenAPI contract with Hey API."
llm_optimized: true
aliases:
---

Loom recommends [`@hey-api/openapi-ts`](https://heyapi.dev/docs/openapi/typescript/get-started)
for TypeScript applications that consume a Loom HTTP service. One generated
OpenAPI contract can drive TypeScript types, Fetch SDK functions, Zod response
validation, and TanStack Query helpers.

This recipe is tested with `@hey-api/openapi-ts` `0.99.0`. Pin the generator and
review its release notes before upgrading because its configuration and output
are external to Loom.

## Emit the compatibility contract

Use Loom's OpenAPI 3.1 compatibility target for downstream client generators:

```go
var _ = API("my_api", func() {
    Meta("openapi:version", "3.1")
})
```

Run Loom generation before the TypeScript generator:

```bash
go tool loom gen example.com/myservice/design
```

The Hey API input is `gen/http/openapi.json`, which identifies itself as
OpenAPI 3.1.1. Loom's default OpenAPI 3.2 output is the richer framework
contract, but the 3.1 target is the continuously tested interoperability
surface for downstream generators.

## Install and configure Hey API

Pin the generator and its TypeScript peer dependency exactly. Install the
runtime packages used by the selected plugins in the frontend application:

```bash
npm install --save-dev --save-exact \
  @hey-api/openapi-ts@0.99.0 typescript@5.9.3
npm install --save-exact \
  @hey-api/client-fetch@0.13.1 @tanstack/react-query@5.101.4 zod@4.4.3
```

Create `openapi-ts.config.ts` in the frontend project:

```ts
import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
  input: "../backend/gen/http/openapi.json",
  output: "src/api/generated",
  plugins: [
    {
      name: "@hey-api/client-fetch",
      baseUrl: false,
      throwOnError: true,
    },
    { name: "@hey-api/typescript" },
    {
      name: "@hey-api/sdk",
      validator: { response: "zod" },
    },
    { name: "zod" },
    {
      name: "@tanstack/react-query",
      queryOptions: true,
      mutationOptions: true,
      queryKeys: { tags: true },
    },
  ],
});
```

Loom operation IDs use `service.method`. Hey API 0.99 already splits operation
IDs on `.` and `/` by default, so no `nestingDelimiters` override is needed.
Loom also emits each service name as an operation tag; tag-based query keys can
therefore invalidate the queries for one service together.

Treat `src/api/generated` as generator-owned and never edit files inside it.
The frontend TypeScript configuration must include the `DOM` and `DOM.Iterable`
libraries because the generated Fetch and query-key helpers use browser APIs.

## Configure the runtime client

Configure the generated client once during application startup, outside the
generated directory:

```ts
import { client } from "./api/generated/client.gen";

client.setConfig({
  baseUrl: import.meta.env.VITE_API_URL,
  credentials: "include",
});
```

`credentials: "include"` is required when the browser calls a different origin
and the Loom API uses `SessionAuth` with `CookieTransport`. Same-origin Fetch
requests include credentials by default, but setting it explicitly keeps the
client correct if deployment topology changes. Configure CORS credentials and
allowed origins on the Loom server as well.

## Handle declared problems

Loom models declared HTTP failures as RFC 9457 Problem Details. When a response
maps to Loom's generated `Problem` schema, Hey API emits per-operation error
maps such as:

```ts
export type SongsGetErrors = {
  401: Problem;
  404: Problem;
};
```

`Problem.detail` is the occurrence-specific human-readable message, while
`Problem.code` is the stable Loom error name intended for program logic.
`Problem.status` carries the runtime HTTP status.

With `throwOnError: true`, catch values are still `unknown`; validate or narrow
the thrown value before reading `code`, `detail`, or `status`. The per-status
maps describe the generated SDK contract but do not turn JavaScript catch
values into a status-discriminated union. Use response-returning mode or a
client interceptor when application logic needs both the transport response
and the parsed problem body.

## Regenerate without drift

Keep backend contract and frontend artifacts in one command:

```json
{
  "scripts": {
    "generate:api": "cd ../backend && go tool loom gen example.com/myservice/design && cd ../frontend && openapi-ts"
  }
}
```

Choose one ownership policy:

- Commit both `gen/http/openapi.json` and `src/api/generated`, then have CI run
  `npm run generate:api` followed by `git diff --exit-code`.
- Commit neither and generate both deterministically before build and test.

Do not check only the OpenAPI file while ignoring generated TypeScript drift.
