package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestPathBuildersEscapeValues checks that the path builders return escaped
// URL paths: string-like values are escaped as one segment, a catch-all value
// keeps its "/" separators, numbers and booleans are formatted as they are,
// and the literal parts of the route are escaped when generating the builder.
// Each value is formatted according to the type of its own parameter, even
// when the route lists the wildcards in another order than the params.
func TestPathBuildersEscapeValues(t *testing.T) {
	root := RunHTTPDSL(t, pathEscapingDSL)
	services := CreateHTTPServices(root)
	paths := filesCode(t, ServerPathFiles(services))

	for _, want := range []string{
		"func ItemPathescPath(id string) string {\n\treturn fmt.Sprintf(\"/items/%v\", loomhttp.EscapePathSegment(id))\n}\n",
		"func FilePathescPath(id int, path string) string {\n\treturn fmt.Sprintf(\"/items/%v/files/%v\", id, loomhttp.EscapePathRemainder(path))\n}\n",
		"func TagsPathescPath(name string, tags []string) string {\n" +
			"\ttagsSlice := make([]string, len(tags))\n" +
			"\tfor i, v := range tags {\n" +
			"\t\ttagsSlice[i] = loomhttp.EscapePathSegment(v)\n" +
			"\t}\n" +
			"\treturn fmt.Sprintf(\"/tags/%v/%v\", loomhttp.EscapePathSegment(name), strings.Join(tagsSlice, \",\"))\n}\n",
		"func OrderPathescPath2(tags []int, name string) string {\n" +
			"\ttagsSlice := make([]string, len(tags))\n" +
			"\tfor i, v := range tags {\n" +
			"\t\ttagsSlice[i] = strconv.FormatInt(int64(v), 10)\n" +
			"\t}\n" +
			"\treturn fmt.Sprintf(\"/y/%v/%v\", strings.Join(tagsSlice, \",\"), loomhttp.EscapePathSegment(name))\n}\n",
		"func FlagsPathescPath(on bool, ratio float64, raw []byte) string {\n\treturn fmt.Sprintf(\"/flags/%v/%v/%v\", on, ratio, loomhttp.EscapePathSegment(string(raw)))\n}\n",
		"func AliasPathescPath(code string) string {\n\treturn fmt.Sprintf(\"/alias/%v\", loomhttp.EscapePathSegment(code))\n}\n",
		"func LiteralPathescPath(id string) string {\n\treturn fmt.Sprintf(\"/my%%20files/100%%25/%v\", loomhttp.EscapePathSegment(id))\n}\n",
		"func StaticPathescPath() string {\n\treturn \"/static%20files\"\n}\n",
		"func PlainPathescPath() string {\n\treturn \"/plain\"\n}\n",
	} {
		assert.Contains(t, paths, want)
	}
}

// TestClientKeepsEscapedPaths checks that the client builds the URL of a
// request from the escaped path of the path builder with RequestURL, and keeps
// the plain URL of a route whose path builder returns an unescaped literal.
func TestClientKeepsEscapedPaths(t *testing.T) {
	root := RunHTTPDSL(t, pathEscapingDSL)
	services := CreateHTTPServices(root)
	client := filesCode(t, ClientFiles("gen", services))

	for _, want := range []string{
		"\tu, err := loomhttp.RequestURL(c.scheme, c.host, ItemPathescPath(id))\n" +
			"\tif err != nil {\n" +
			"\t\treturn nil, loomhttp.ErrInvalidURL(\"pathesc\", \"item\", u.String(), err)\n" +
			"\t}\n",
		"\tu, err := loomhttp.RequestURL(c.scheme, c.host, StaticPathescPath())\n",
		"\tu := &url.URL{Scheme: c.scheme, Host: c.host, Path: PlainPathescPath()}\n",
	} {
		assert.Contains(t, client, want)
	}
	require.NotContains(t, client, "Path: ItemPathescPath(")
}

func pathEscapingDSL() {
	code := Type("Code", String)
	Service("pathesc", func() {
		Method("item", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			HTTP(func() {
				GET("/items/{id}")
			})
		})
		Method("file", func() {
			Payload(func() {
				Attribute("id", Int)
				Attribute("path", String)
				Required("id", "path")
			})
			HTTP(func() {
				GET("/items/{id}/files/{*path}")
			})
		})
		Method("tags", func() {
			Payload(func() {
				Attribute("tags", ArrayOf(String))
				Attribute("name", String)
				Required("tags", "name")
			})
			HTTP(func() {
				GET("/tags/{name}/{tags}")
			})
		})
		Method("order", func() {
			Payload(func() {
				Attribute("tags", ArrayOf(Int))
				Attribute("name", String)
				Required("tags", "name")
			})
			HTTP(func() {
				GET("/x/{name}/{tags}")
				GET("/y/{tags}/{name}")
			})
		})
		Method("flags", func() {
			Payload(func() {
				Attribute("on", Boolean)
				Attribute("ratio", Float64)
				Attribute("raw", Bytes)
				Required("on", "ratio", "raw")
			})
			HTTP(func() {
				GET("/flags/{on}/{ratio}/{raw}")
			})
		})
		Method("alias", func() {
			Payload(func() {
				Attribute("code", code)
				Required("code")
			})
			HTTP(func() {
				GET("/alias/{code}")
			})
		})
		Method("literal", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			HTTP(func() {
				GET("/my files/100%/{id}")
			})
		})
		Method("static", func() {
			HTTP(func() {
				GET("/static files")
			})
		})
		Method("plain", func() {
			HTTP(func() {
				GET("/plain")
			})
		})
	})
}
