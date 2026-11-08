package openapiimport

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

func partitionServers(version string, servers []Server) ([]Server, Diagnostics) {
	retained := make([]Server, 0, len(servers))
	var diagnostics Diagnostics
	for index, server := range servers {
		path := server.sourcePath
		if path == "" {
			path = fmt.Sprintf("#/servers/%d", index)
		}
		reason := server.sourceError
		parsed, err := url.Parse(server.URL)
		switch {
		case reason != "":
		case err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https"):
			reason = "root server URL must be an absolute HTTP or HTTPS URL with a host"
		case parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.ContainsAny(server.URL, "{}#"):
			reason = "root server userinfo, query, fragment and template variables are not in the strict import subset"
		case strings.HasPrefix(version, "3.2.") && (server.Name == nil || *server.Name == ""):
			reason = "OpenAPI 3.2 root servers without a nonempty authored name are not in the strict import subset"
		case !strings.HasPrefix(version, "3.2.") && server.Name != nil:
			reason = "root server name requires OpenAPI 3.2"
		}
		if reason != "" {
			diagnostics = append(diagnostics, Diagnostic{Code: "servers", Path: path, Message: reason})
		} else {
			retained = append(retained, server)
		}
	}
	return retained, diagnostics
}

func (a *analyzer) servers(sources []*v3.Server) []Server {
	servers := make([]Server, 0, len(sources))
	for index, source := range sources {
		server := Server{sourcePath: fmt.Sprintf("#/servers/%d", index)}
		if source == nil {
			server.sourceError = "root server is nil"
			servers = append(servers, server)
			continue
		}
		server.URL = source.URL
		low := source.GoLow()
		if low == nil {
			server.sourceError = "root server source presence is unavailable"
		} else {
			if !low.Name.IsEmpty() {
				name := source.Name
				server.Name = &name
			}
			if !low.Description.IsEmpty() {
				description := source.Description
				server.Description = &description
			}
		}
		if orderedmap.Len(source.Variables) > 0 || orderedmap.Len(source.Extensions) > 0 {
			server.sourceError = "root server variables and extensions are not in the strict import subset"
		}
		servers = append(servers, server)
	}
	return servers
}

func (r *renderer) servers() {
	for index, server := range r.document.Servers {
		name := fmt.Sprintf("ImportedServer%d", index+1)
		if server.Name != nil {
			name = *server.Name
		}
		r.open("Server(%q, func()", name)
		r.open("Host(%q, func()", "imported")
		if server.Description != nil {
			r.line("Description(%q)", *server.Description)
		}
		r.line("URI(%q)", server.URL)
		r.close()
		r.close()
	}
}
