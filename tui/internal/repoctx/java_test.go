package repoctx

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

const javaOrderFixture = `package com.acme.order;

import org.springframework.web.bind.annotation.*;
import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Autowired;

@RestController
@RequestMapping("/orders")
public class OrderController {
    private final OrderService service;
    @Autowired private AuditRepository auditRepository;

    public OrderController(OrderService service) {
        this.service = service;
    }

    @PostMapping
    public Order createOrder(OrderRequest request) {
        auditRepository.touch();
        return service.createOrder(request);
    }

    @GetMapping("/{id}")
    public Order getOrder(String id) { return service.getOrder(id); }

    public Order getOrder(long id) { return service.getOrder(String.valueOf(id)); }
}

class FakeService {
    // @Service class PretendService {}
    String example = "@RestController class Phantom {}";
    String text = """@Repository class TextBlockFake {}""";
    void unknown() { mystery.call(); }
}
`

func TestJavaSpringExtractsDeclarationsAnnotationsAndTypedRelations(t *testing.T) {
	f := ExtractJava("src/main/java/com/acme/order/OrderController.java", []byte(javaOrderFixture))
	if f.Package != "com.acme.order" {
		t.Fatalf("package=%q", f.Package)
	}
	if len(f.Imports) != 3 {
		t.Fatalf("imports=%v", f.Imports)
	}

	controller := findSymbol(t, f.Symbols, "OrderController", SymbolClass)
	if controller.Role != "rest-controller" {
		t.Fatalf("role=%q annotations=%v", controller.Role, controller.Annotations)
	}
	requireAnnotation(t, controller, "RestController")
	requireAnnotation(t, controller, "RequestMapping")

	ctor := findSymbol(t, f.Symbols, "OrderController", SymbolConstructor)
	if ctor.Signature != "OrderController(OrderService)" {
		t.Fatalf("ctor signature=%q", ctor.Signature)
	}

	create := findSymbol(t, f.Symbols, "createOrder", SymbolMethod)
	requireAnnotation(t, create, "PostMapping")
	if create.Type != "Order" {
		t.Fatalf("return type=%q", create.Type)
	}

	if !hasCandidate(f, RelationInjects, controller.ID, "OrderService", "") {
		t.Fatalf("missing constructor injection candidate: %+v", f.candidates)
	}
	if !hasCandidate(f, RelationInjects, controller.ID, "AuditRepository", "") {
		t.Fatalf("missing @Autowired field injection candidate: %+v", f.candidates)
	}
	if !hasCandidate(f, RelationCalls, create.ID, "OrderService", "createOrder") {
		t.Fatalf("missing typed receiver call: %+v", f.candidates)
	}
}

func TestJavaSpringRolesRequireAnnotationEvidence(t *testing.T) {
	f := ExtractJava("FakeService.java", []byte(javaOrderFixture))
	fake := findSymbol(t, f.Symbols, "FakeService", SymbolClass)
	if fake.Role != "" {
		t.Fatalf("FakeService inferred role from suffix: %q", fake.Role)
	}
	for _, bad := range []string{"PretendService", "Phantom", "TextBlockFake"} {
		for _, s := range f.Symbols {
			if s.Name == bad {
				t.Fatalf("symbol created from comment/string/text block: %+v", s)
			}
		}
	}
}

func TestJavaOverloadsHaveDistinctStableIDs(t *testing.T) {
	first := ExtractJava("OrderController.java", []byte(javaOrderFixture))
	second := ExtractJava("OrderController.java", []byte(javaOrderFixture))
	var ids1, ids2 []string
	for _, f := range []SourceFile{first, second} {
		ids := []string{}
		for _, s := range f.Symbols {
			if s.Name == "getOrder" {
				ids = append(ids, s.ID)
			}
		}
		sort.Strings(ids)
		if len(ids) != 2 {
			t.Fatalf("getOrder overload count=%d symbols=%+v", len(ids), f.Symbols)
		}
		if f.Path == first.Path {
			if ids1 == nil {
				ids1 = ids
			} else {
				ids2 = ids
			}
		}
	}
	if ids1[0] == ids1[1] {
		t.Fatal("overloaded IDs collide")
	}
	if !reflect.DeepEqual(ids1, ids2) {
		t.Fatalf("IDs unstable: %v vs %v", ids1, ids2)
	}
}

func TestJavaUnknownReceiverRemainsUnresolved(t *testing.T) {
	f := ExtractJava("FakeService.java", []byte(javaOrderFixture))
	joined := strings.Join(f.Unresolved, "\n")
	if !strings.Contains(joined, "mystery.call") {
		t.Fatalf("unknown receiver not unresolved: %v", f.Unresolved)
	}
	for _, c := range f.candidates {
		if c.receiver == "mystery" {
			t.Fatalf("unknown receiver promoted: %+v", c)
		}
	}
}

func findSymbol(t *testing.T, symbols []Symbol, name string, kind SymbolKind) Symbol {
	t.Helper()
	for _, s := range symbols {
		if s.Name == name && s.Kind == kind {
			return s
		}
	}
	t.Fatalf("missing %s %s in %+v", kind, name, symbols)
	return Symbol{}
}
func requireAnnotation(t *testing.T, s Symbol, ann string) {
	t.Helper()
	for _, a := range s.Annotations {
		if a == ann {
			return
		}
	}
	t.Fatalf("%s missing @%s: %v", s.Name, ann, s.Annotations)
}
func hasCandidate(f SourceFile, kind RelationKind, from, targetType, targetMethod string) bool {
	for _, c := range f.candidates {
		if c.kind == kind && c.from == from && c.targetType == targetType && c.targetMethod == targetMethod {
			return true
		}
	}
	return false
}
