package repoctx

import (
	"strings"
	"testing"
)

const goFixture = `package orders
import (
    "context"
    "fmt"
)
type Service interface { Create(context.Context, string) error }
type Handler struct { svc Service }
func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Create(ctx context.Context, id string) error {
    err := h.svc.Create(ctx, id)
    fmt.Println(id)
    return err
}
func helper() {}
`

func TestGoExtractsASTStructureAndStableLines(t *testing.T) {
	f := ExtractGo("internal/orders/handler.go", []byte(goFixture))
	if f.Package != "orders" {
		t.Fatalf("package=%q", f.Package)
	}
	if len(f.Imports) != 2 || f.Imports[0] != "context" || f.Imports[1] != "fmt" {
		t.Fatalf("imports=%v", f.Imports)
	}
	findSymbol(t, f.Symbols, "Service", SymbolInterface)
	findSymbol(t, f.Symbols, "Handler", SymbolType)
	fn := findSymbol(t, f.Symbols, "NewHandler", SymbolFunction)
	if fn.StartLine != 8 {
		t.Fatalf("NewHandler line=%d", fn.StartLine)
	}
	method := findSymbol(t, f.Symbols, "Create", SymbolMethod)
	if method.Owner != "Handler" || method.StartLine != 9 {
		t.Fatalf("method=%+v", method)
	}
	if !hasCandidate(f, RelationCalls, method.ID, "Service", "Create") {
		t.Fatalf("missing deterministic selector call: %+v", f.candidates)
	}
}

func TestGoInvalidFileDegradesToUnresolved(t *testing.T) {
	f := ExtractGo("broken.go", []byte("package x\nfunc nope( {"))
	if len(f.Unresolved) == 0 {
		t.Fatal("invalid Go should preserve parse evidence")
	}
	if !strings.Contains(strings.Join(f.Unresolved, " "), "broken.go") {
		t.Fatalf("unresolved=%v", f.Unresolved)
	}
}
