package repoctx

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRepoMapRendersCompactSectionsAndFiltersSensitivePaths(t *testing.T) {
	m := RepoMap{
		Summary:      "task relevant",
		Files:        []string{"src/OrderController.java", ".env"},
		Symbols:      []Symbol{{ID: "a", Path: "src/OrderController.java", Owner: "OrderController", Name: "createOrder", Signature: "createOrder(OrderRequest)", StartLine: 42}},
		Relations:    []Relation{{From: "a", To: "b", Kind: RelationCalls, Evidence: "typed receiver"}},
		Unresolved:   []string{"ambiguous B#work", ".env: secret evidence"},
		ChangedFiles: []string{"src/OrderController.java", ".env"},
		maxChars:     4000,
	}
	got := m.Render()
	for _, want := range []string{"REPO CONTEXT", "Relevant files", "Relevant symbols", "Unresolved/ambiguous evidence", "Changed files"} {
		if !strings.Contains(got, want) {
			t.Fatalf("map missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, ".env") {
		t.Fatalf("sensitive environment path leaked:\n%s", got)
	}
	if strings.Contains(got, "class OrderController") {
		t.Fatalf("map should contain compact signatures, not full file dumps:\n%s", got)
	}
}

func TestRepoMapBudgetAndDeterminism(t *testing.T) {
	base := RepoMap{Summary: "large deterministic fixture"}
	for i := 0; i < 400; i++ {
		path := fmt.Sprintf("src/%03d/OrderComponent.java", i)
		base.Files = append(base.Files, path)
		base.Symbols = append(base.Symbols, Symbol{ID: fmt.Sprintf("s%03d", i), Path: path, Owner: "OrderComponent", Name: fmt.Sprintf("method%03d", i), StartLine: i + 1})
		base.Unresolved = append(base.Unresolved, fmt.Sprintf("candidate-%03d remains unresolved", i))
	}
	for _, budget := range []int{4000, 8000, 12000} {
		a := base
		a.maxChars = budget
		first := a.Render()
		if utf8.RuneCountInString(first) > budget {
			t.Fatalf("budget=%d exceeded with %d chars", budget, utf8.RuneCountInString(first))
		}
		if !a.Truncated {
			t.Fatalf("budget=%d should truncate large fixture", budget)
		}
		if !strings.Contains(first, "src/000/OrderComponent.java") {
			t.Fatalf("budget=%d dropped highest-ranked evidence first", budget)
		}
		b := base
		b.maxChars = budget
		second := b.Render()
		if first != second {
			t.Fatalf("budget=%d output is not deterministic", budget)
		}
	}
}

func TestServiceBuildMapIncludesRelevantJavaChain(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "src/OrderController.java", `package com.acme.order; import org.springframework.web.bind.annotation.RestController;
@RestController public class OrderController { private final OrderService service; public OrderController(OrderService service){this.service=service;} public Order createOrder(OrderRequest r){return service.createOrder(r);} }`)
	writeRepoFile(t, root, "src/OrderService.java", `package com.acme.order; import org.springframework.stereotype.Service;
@Service public class OrderService { private final OrderRepository repository; public OrderService(OrderRepository repository){this.repository=repository;} public Order createOrder(OrderRequest r){return repository.save(new Order());} }`)
	writeRepoFile(t, root, "src/OrderRepository.java", `package com.acme.order; import org.springframework.stereotype.Repository;
@Repository public interface OrderRepository { Order save(Order order); }`)
	writeRepoFile(t, root, ".env", "PRIVATE_TOKEN=do-not-render")

	m, err := NewService(root).BuildMap(context.Background(), Query{Text: "OrderController#createOrder", MaxChars: 8000})
	if err != nil {
		t.Fatal(err)
	}
	got := m.Render()
	for _, want := range []string{"OrderController#createOrder", "OrderService#createOrder", "OrderRepository#save"} {
		if !strings.Contains(got, want) {
			t.Fatalf("repo map missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "PRIVATE_TOKEN") || strings.Contains(got, ".env") {
		t.Fatalf("repo map leaked environment data:\n%s", got)
	}
}
