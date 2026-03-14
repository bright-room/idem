package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSelectedIfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mask     int
		wantName []string
	}{
		{"single bit 001", 0b001, []string{"Flusher"}},
		{"single bit 010", 0b010, []string{"Hijacker"}},
		{"single bit 100", 0b100, []string{"ReaderFrom"}},
		{"two bits 011", 0b011, []string{"Flusher", "Hijacker"}},
		{"two bits 101", 0b101, []string{"Flusher", "ReaderFrom"}},
		{"two bits 110", 0b110, []string{"Hijacker", "ReaderFrom"}},
		{"all bits 111", 0b111, []string{"Flusher", "Hijacker", "ReaderFrom"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := selectedIfaces(tt.mask)
			if len(got) != len(tt.wantName) {
				t.Fatalf("selectedIfaces(%03b) returned %d interfaces, want %d", tt.mask, len(got), len(tt.wantName))
			}
			for i, ifc := range got {
				if ifc.Name != tt.wantName[i] {
					t.Errorf("selectedIfaces(%03b)[%d].Name = %q, want %q", tt.mask, i, ifc.Name, tt.wantName[i])
				}
			}
		})
	}
}

func TestBuildTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{"single interface", []iface{{Name: "Flusher"}}, "responseRecorderFlusher"},
		{"two interfaces", []iface{{Name: "Flusher"}, {Name: "Hijacker"}}, "responseRecorderFlusherHijacker"},
		{"all interfaces", []iface{{Name: "Flusher"}, {Name: "Hijacker"}, {Name: "ReaderFrom"}}, "responseRecorderFlusherHijackerReaderFrom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildTypeName(tt.selected); got != tt.want {
				t.Errorf("buildTypeName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{
			"one interface",
			[]iface{{Type: "http.Flusher"}},
			"delegates http.Flusher to the underlying ResponseWriter.",
		},
		{
			"two interfaces",
			[]iface{{Type: "http.Flusher"}, {Type: "http.Hijacker"}},
			"delegates both http.Flusher and http.Hijacker\n// to the underlying ResponseWriter.",
		},
		{
			"three interfaces",
			[]iface{{Type: "http.Flusher"}, {Type: "http.Hijacker"}, {Type: "io.ReaderFrom"}},
			"delegates http.Flusher, http.Hijacker,\n// and io.ReaderFrom to the underlying ResponseWriter.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildComment(tt.selected); got != tt.want {
				t.Errorf("buildComment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{"single", []iface{{CheckName: "canFlush"}}, "canFlush"},
		{"two", []iface{{CheckName: "canFlush"}, {CheckName: "canHijack"}}, "canFlush && canHijack"},
		{"three", []iface{{CheckName: "canFlush"}, {CheckName: "canHijack"}, {CheckName: "canReadFrom"}}, "canFlush && canHijack && canReadFrom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildConditions(tt.selected); got != tt.want {
				t.Errorf("buildConditions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{"single", []iface{{VarName: "flusher"}}, "flusher"},
		{"two", []iface{{VarName: "flusher"}, {VarName: "hijacker"}}, "flusher, hijacker"},
		{"three", []iface{{VarName: "flusher"}, {VarName: "hijacker"}, {VarName: "readerFrom"}}, "flusher, hijacker, readerFrom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildArgs(tt.selected); got != tt.want {
				t.Errorf("buildArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ifaces []iface
		want   []string
	}{
		{
			"deduplicates imports",
			[]iface{
				{Import: "net/http"},
				{Import: "net/http"},
				{Import: "io"},
			},
			[]string{"net/http", "io"},
		},
		{
			"all unique",
			[]iface{
				{Import: "net/http"},
				{Import: "io"},
			},
			[]string{"net/http", "io"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := collectImports(tt.ifaces)
			if len(got) != len(tt.want) {
				t.Fatalf("collectImports() returned %d imports, want %d", len(got), len(tt.want))
			}
			for i, imp := range got {
				if imp != tt.want[i] {
					t.Errorf("collectImports()[%d] = %q, want %q", i, imp, tt.want[i])
				}
			}
		})
	}
}

func TestBuildMockName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{"single", []iface{{Name: "Flusher"}}, "flusherWriter"},
		{"two", []iface{{Name: "Flusher"}, {Name: "Hijacker"}}, "flusherHijackerWriter"},
		{"all", []iface{{Name: "Flusher"}, {Name: "Hijacker"}, {Name: "ReaderFrom"}}, "flusherHijackerReaderFromWriter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildMockName(tt.selected); got != tt.want {
				t.Errorf("buildMockName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildMockComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selected []iface
		want     string
	}{
		{
			"one interface",
			[]iface{{Type: "http.Flusher"}},
			"implements http.ResponseWriter and http.Flusher.",
		},
		{
			"two interfaces",
			[]iface{{Type: "http.Flusher"}, {Type: "http.Hijacker"}},
			"implements http.ResponseWriter, http.Flusher,\n// and http.Hijacker.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildMockComment(tt.selected); got != tt.want {
				t.Errorf("buildMockComment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMockFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ifaceName string
		want      string
	}{
		{"Flusher", "Flusher", "flushed"},
		{"Hijacker", "Hijacker", "hijacked"},
		{"ReaderFrom", "ReaderFrom", "readFrom"},
		{"unknown falls back to lowercase", "Pusher", "pusher"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := mockFieldName(tt.ifaceName); got != tt.want {
				t.Errorf("mockFieldName(%q) = %q, want %q", tt.ifaceName, got, tt.want)
			}
		})
	}
}

func TestGenerateCode_Structure(t *testing.T) {
	t.Parallel()

	n := len(interfaces)
	var combos []combo
	var cases []switchCase

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

	src, err := renderToBytes(codeTemplate, data)
	if err != nil {
		t.Fatalf("renderToBytes() error: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "recorder_gen.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated code does not parse: %v", err)
	}

	if f.Name.Name != "idem" {
		t.Errorf("package name = %q, want %q", f.Name.Name, "idem")
	}

	typeNames := map[string]bool{}
	funcNames := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.TypeSpec:
			typeNames[decl.Name.Name] = true
		case *ast.FuncDecl:
			funcNames[decl.Name.Name] = true
		}
		return true
	})

	wantTypes := []string{
		"responseRecorderFlusherHijackerReaderFrom",
		"responseRecorderHijackerReaderFrom",
		"responseRecorderFlusherReaderFrom",
		"responseRecorderFlusherHijacker",
		"responseRecorderReaderFrom",
		"responseRecorderHijacker",
		"responseRecorderFlusher",
	}
	if len(typeNames) != len(wantTypes) {
		t.Errorf("found %d type definitions, want %d", len(typeNames), len(wantTypes))
	}
	for _, name := range wantTypes {
		if !typeNames[name] {
			t.Errorf("missing type definition: %s", name)
		}
	}

	if !funcNames["newResponseRecorder"] {
		t.Error("missing function: newResponseRecorder")
	}
}

func TestGenerateTestCode_Structure(t *testing.T) {
	t.Parallel()

	n := len(interfaces)
	var mocks []testMock

	mocks = append(mocks, testMock{
		TypeName: "plainWriter",
		Comment:  "implements only http.ResponseWriter.",
	})

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

	src, err := renderToBytes(testTemplate, data)
	if err != nil {
		t.Fatalf("renderToBytes() error: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "recorder_gen_test.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated test code does not parse: %v", err)
	}

	typeNames := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			typeNames[ts.Name.Name] = true
		}
		return true
	})

	// plainWriter + 7 combinations = 8
	wantMockCount := 8
	if len(typeNames) != wantMockCount {
		t.Errorf("found %d mock types, want %d", len(typeNames), wantMockCount)
	}

	if !typeNames["plainWriter"] {
		t.Error("missing mock type: plainWriter")
	}
}

func TestCombinationCount_And_Order(t *testing.T) {
	t.Parallel()

	n := len(interfaces)
	wantCount := (1 << n) - 1 // 2^3 - 1 = 7

	var masks []int
	for mask := (1 << n) - 1; mask >= 1; mask-- {
		masks = append(masks, mask)
	}

	if len(masks) != wantCount {
		t.Fatalf("got %d combinations, want %d", len(masks), wantCount)
	}

	// Verify descending bitmask order so the most-specific cases appear first
	// in the generated switch statement.
	for i := 1; i < len(masks); i++ {
		if masks[i] >= masks[i-1] {
			t.Errorf("mask[%d]=%d >= mask[%d]=%d; want strictly descending", i, masks[i], i-1, masks[i-1])
		}
	}
}
