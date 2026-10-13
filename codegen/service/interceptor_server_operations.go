//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func addServerOperationFields(group *jen.Group, plan *serverStreamPlan) {
	for _, op := range plan.operations {
		if op.action == streamDelegate || op.contextual != "" {
			continue
		}
		group.Id(op.callback()).Func().ParamsFunc(func(params *jen.Group) {
			for _, param := range op.parameters {
				params.Add(codegen.TypeRef(param.ref))
			}
		}).ParamsFunc(func(results *jen.Group) {
			for _, ref := range op.results {
				results.Add(codegen.TypeRef(ref))
			}
		})
	}
}

func addServerOperationMethods(stmt *jen.Statement, stream *StreamInterceptorData) {
	for _, op := range stream.serverPlan.operations {
		stmt.Line()
		codegen.Doc(stmt, op.description)
		signature := stmt.Func().Params(jen.Id("w").Op("*").Id(stream.WrapperName)).Id(op.name).
			ParamsFunc(func(params *jen.Group) {
				for _, param := range op.parameters {
					params.Id(param.name).Add(codegen.TypeRef(param.ref))
				}
			})
		if len(op.results) > 0 {
			signature.ParamsFunc(func(results *jen.Group) {
				for _, ref := range op.results {
					results.Add(codegen.TypeRef(ref))
				}
			})
		}
		signature.BlockFunc(func(group *jen.Group) {
			var args []string
			for _, param := range op.parameters {
				args = append(args, param.name)
			}
			prefix := ""
			if len(op.results) > 0 {
				prefix = "return "
			}
			if op.contextual != "" {
				args = append([]string{"w.ctx"}, args...)
				addRawGroup(group, prefix+"w."+op.contextual+"("+strings.Join(args, ", ")+")")
				return
			}
			call := "(" + strings.Join(args, ", ") + ")"
			if op.action != streamDelegate {
				addRawGroup(group, "if w."+op.callback()+" == nil {\n"+prefix+"w.stream."+op.name+call+"\n}\n"+prefix+"w."+op.callback()+call)
				return
			}
			addRawGroup(group, prefix+"w.stream."+op.name+call)
		})
	}
	stmt.Line()
}

func renderServerMessageInterception(b *strings.Builder, service string, interceptor *InterceptorData, method *MethodInterceptorData) {
	stream := method.ServerStream
	plan := stream.serverPlan
	if plan.incoming && interceptor.HasStreamingPayloadAccess {
		message := "req"
		if plan.endpoint != "" {
			message = "req.(*" + plan.endpoint + ").Payload"
		}
		fmt.Fprintf(b, "message, err := i.%s(ctx, &%sInfo{\nservice: %q,\nmethod: %q,\ncallType: loom.InterceptorStreamingRecv,\n}, func(context.Context, any) (any, error) {\nreturn %s, nil\n})\nif err != nil {\nreturn nil, err\n}\n", interceptor.Name, interceptor.Name, service, method.MethodName, message)
		if plan.endpoint != "" {
			fmt.Fprintf(b, "%s, _ = message.(%s)\n", message, method.StreamingPayloadRef)
		} else {
			b.WriteString("req = message\n")
		}
	}
	if plan.endpoint == "" {
		return
	}
	fmt.Fprintf(b, "stream := req.(*%s).Stream\n", plan.endpoint)
	fmt.Fprintf(b, "req.(*%s).Stream = &%s{\nctx: ctx,\n", plan.endpoint, stream.WrapperName)
	for _, op := range plan.operations {
		if op.contextual != "" {
			continue
		}
		switch {
		case op.action == streamSend && interceptor.HasStreamingResultAccess:
			renderServerSendInterception(b, service, interceptor, method, op)
		case op.action == streamReceive && interceptor.HasStreamingPayloadAccess:
			fmt.Fprintf(b, "%s: func(ctx context.Context) (%s, error) {\n", op.callback(), stream.RecvTypeRef)
			fmt.Fprintf(b, "info := &%sInfo{\nservice: %q,\nmethod: %q,\ncallType: loom.InterceptorStreamingRecv,\n}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(b, "res, err := i.%s(ctx, info, func(ctx context.Context, _ any) (any, error) {\nreturn stream.%s(ctx)\n})\ncastRes, _ := res.(%s)\nreturn castRes, err\n},\n", interceptor.Name, op.name, stream.RecvTypeRef)
		}
	}
	b.WriteString("stream: stream,\n}\n")
}

func renderServerSendInterception(b *strings.Builder, service string, interceptor *InterceptorData, method *MethodInterceptorData, op serverStreamOperation) {
	// SSE uses its event interface in the operation signature; the typed accessor
	// uses the canonical concrete result, which the event contract admits.
	fmt.Fprintf(b, "%s: func(ctx context.Context, req %s) error {\n", op.callback(), op.parameters[len(op.parameters)-1].ref)
	fmt.Fprintf(b, "info := &%sInfo{\nservice: %q,\nmethod: %q,\ncallType: loom.InterceptorStreamingSend,\nrawPayload: req,\n}\n", interceptor.Name, service, method.MethodName)
	fmt.Fprintf(b, "_, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\ncastReq, _ := req.(%s)\nreturn nil, stream.%s(ctx, castReq)\n})\nreturn err\n},\n", interceptor.Name, method.ServerStream.SendTypeRef, op.name)
}
