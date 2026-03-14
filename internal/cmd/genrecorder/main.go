// genrecorder generates combination types for responseRecorder that delegate
// optional http.ResponseWriter interfaces (http.Flusher, http.Hijacker, io.ReaderFrom)
// to the underlying writer.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"
)

type iface struct {
	Name      string // e.g., "Flusher"
	Package   string // e.g., "http"
	Import    string // e.g., "net/http"
	Type      string // e.g., "http.Flusher"
	VarName   string // e.g., "flusher"
	CheckName string // e.g., "canFlush"
}

type combo struct {
	TypeName   string
	Comment    string
	Interfaces []iface
}

type switchCase struct {
	Conditions string
	TypeName   string
	Args       string
}

type templateData struct {
	Combos  []combo
	Cases   []switchCase
	Ifaces  []iface
	Imports []string
}

// testMock represents a mock writer type for test code generation.
type testMock struct {
	TypeName   string
	Comment    string
	Interfaces []iface
	Fields     []mockField
}

type mockField struct {
	Name string
	Type string
}

type testTemplateData struct {
	Mocks   []testMock
	Imports []string
}

var interfaces = []iface{
	{Name: "Flusher", Package: "http", Import: "net/http", Type: "http.Flusher", VarName: "flusher", CheckName: "canFlush"},
	{Name: "Hijacker", Package: "http", Import: "net/http", Type: "http.Hijacker", VarName: "hijacker", CheckName: "canHijack"},
	{Name: "ReaderFrom", Package: "io", Import: "io", Type: "io.ReaderFrom", VarName: "readerFrom", CheckName: "canReadFrom"},
}

func main() {
	outFile := flag.String("out", "recorder_gen.go", "output file for generated code")
	testOutFile := flag.String("test-out", "recorder_gen_test.go", "output file for generated test code")
	flag.Parse()

	if err := generateCode(*outFile); err != nil {
		fmt.Fprintf(os.Stderr, "genrecorder: %v\n", err)
		os.Exit(1)
	}

	if err := generateTestCode(*testOutFile); err != nil {
		fmt.Fprintf(os.Stderr, "genrecorder: %v\n", err)
		os.Exit(1)
	}
}

func generateCode(outFile string) error {
	n := len(interfaces)
	var combos []combo
	var cases []switchCase

	// Generate all combinations using bitmask (from most bits to fewest).
	for mask := (1 << n) - 1; mask >= 1; mask-- {
		selected := selectedIfaces(mask)
		typeName := buildTypeName(selected)
		comment := buildComment(selected)
		combos = append(combos, combo{
			TypeName:   typeName,
			Comment:    comment,
			Interfaces: selected,
		})
		cases = append(cases, switchCase{
			Conditions: buildConditions(selected),
			TypeName:   typeName,
			Args:       buildArgs(selected),
		})
	}

	imports := collectImports(interfaces)
	data := templateData{
		Combos:  combos,
		Cases:   cases,
		Ifaces:  interfaces,
		Imports: imports,
	}

	return renderToFile(outFile, codeTemplate, data)
}

func generateTestCode(outFile string) error {
	n := len(interfaces)
	var mocks []testMock

	// plain writer (no interfaces)
	mocks = append(mocks, testMock{
		TypeName: "plainWriter",
		Comment:  "implements only http.ResponseWriter.",
	})

	// All combinations.
	for mask := 1; mask < (1 << n); mask++ {
		selected := selectedIfaces(mask)
		mockName := buildMockName(selected)
		comment := buildMockComment(selected)
		var fields []mockField
		for _, ifc := range selected {
			fields = append(fields, mockField{
				Name: mockFieldName(ifc.Name),
				Type: "bool",
			})
		}
		mocks = append(mocks, testMock{
			TypeName:   mockName,
			Comment:    comment,
			Interfaces: selected,
			Fields:     fields,
		})
	}

	data := testTemplateData{
		Mocks:   mocks,
		Imports: []string{"bufio", "io", "net", "net/http"},
	}

	return renderToFile(outFile, testTemplate, data)
}

func selectedIfaces(mask int) []iface {
	var selected []iface
	for i, ifc := range interfaces {
		if mask&(1<<i) != 0 {
			selected = append(selected, ifc)
		}
	}
	return selected
}

func buildTypeName(selected []iface) string {
	var sb strings.Builder
	sb.WriteString("responseRecorder")
	for _, ifc := range selected {
		sb.WriteString(ifc.Name)
	}
	return sb.String()
}

func buildComment(selected []iface) string {
	names := make([]string, len(selected))
	for i, ifc := range selected {
		names[i] = ifc.Type
	}
	switch len(names) {
	case 1:
		return fmt.Sprintf("delegates %s to the underlying ResponseWriter.", names[0])
	case 2:
		return fmt.Sprintf("delegates both %s and %s\n// to the underlying ResponseWriter.", names[0], names[1])
	default:
		return fmt.Sprintf("delegates %s,\n// and %s to the underlying ResponseWriter.",
			strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
	}
}

func buildConditions(selected []iface) string {
	conds := make([]string, len(selected))
	for i, ifc := range selected {
		conds[i] = ifc.CheckName
	}
	return strings.Join(conds, " && ")
}

func buildArgs(selected []iface) string {
	args := make([]string, len(selected))
	for i, ifc := range selected {
		args[i] = ifc.VarName
	}
	return strings.Join(args, ", ")
}

func collectImports(ifaces []iface) []string {
	seen := make(map[string]bool)
	var imports []string
	for _, ifc := range ifaces {
		if !seen[ifc.Import] {
			seen[ifc.Import] = true
			imports = append(imports, ifc.Import)
		}
	}
	return imports
}

func buildMockName(selected []iface) string {
	var sb strings.Builder
	for i, ifc := range selected {
		if i == 0 {
			sb.WriteString(strings.ToLower(ifc.Name[:1]) + ifc.Name[1:])
		} else {
			sb.WriteString(ifc.Name)
		}
	}
	sb.WriteString("Writer")
	return sb.String()
}

func buildMockComment(selected []iface) string {
	names := make([]string, len(selected))
	for i, ifc := range selected {
		names[i] = ifc.Type
	}
	allNames := append([]string{"http.ResponseWriter"}, names...)
	switch len(allNames) {
	case 2:
		return fmt.Sprintf("implements %s and %s.", allNames[0], allNames[1])
	default:
		return fmt.Sprintf("implements %s,\n// and %s.",
			strings.Join(allNames[:len(allNames)-1], ", "), allNames[len(allNames)-1])
	}
}

func mockFieldName(ifaceName string) string {
	switch ifaceName {
	case "Flusher":
		return "flushed"
	case "Hijacker":
		return "hijacked"
	case "ReaderFrom":
		return "readFrom"
	default:
		return strings.ToLower(ifaceName[:1]) + ifaceName[1:]
	}
}

// renderToBytes executes the template with the given data and returns gofmt-formatted source.
func renderToBytes(tmplText string, data any) ([]byte, error) {
	tmpl, err := template.New("gen").Parse(tmplText)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n\nGenerated code:\n%s", err, buf.String())
	}

	return formatted, nil
}

func renderToFile(filename string, tmplText string, data any) error {
	formatted, err := renderToBytes(tmplText, data)
	if err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}

	return os.WriteFile(filename, formatted, 0o644)
}

var codeTemplate = `// Code generated by genrecorder; DO NOT EDIT.

package idem

import (
{{- range .Imports}}
	"{{.}}"
{{- end}}
)
{{range .Combos}}
// {{.TypeName}} {{.Comment}}
type {{.TypeName}} struct {
	*responseRecorder
{{- range .Interfaces}}
	{{.Type}}
{{- end}}
}
{{end}}
func newResponseRecorder(w http.ResponseWriter) http.ResponseWriter {
	rec := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
{{range .Ifaces}}
	{{.VarName}}, {{.CheckName}} := w.({{.Type}})
{{- end}}

	switch {
{{- range .Cases}}
	case {{.Conditions}}:
		return &{{.TypeName}}{rec, {{.Args}}}
{{- end}}
	default:
		return rec
	}
}
`

var testTemplate = `// Code generated by genrecorder; DO NOT EDIT.

package idem

import (
{{- range .Imports}}
	"{{.}}"
{{- end}}
)
{{range $mock := .Mocks}}
// {{$mock.TypeName}} {{$mock.Comment}}
type {{$mock.TypeName}} struct {
	http.ResponseWriter
{{- range $mock.Fields}}
	{{.Name}} {{.Type}}
{{- end}}
}
{{range $ifc := $mock.Interfaces}}{{if eq $ifc.Name "Flusher"}}
func (w *{{$mock.TypeName}}) Flush() {
	w.flushed = true
}
{{end}}{{if eq $ifc.Name "Hijacker"}}
func (w *{{$mock.TypeName}}) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}
{{end}}{{if eq $ifc.Name "ReaderFrom"}}
func (w *{{$mock.TypeName}}) ReadFrom(r io.Reader) (int64, error) {
	w.readFrom = true
	return io.Copy(w.ResponseWriter, r)
}
{{end}}{{end}}{{end}}
`
