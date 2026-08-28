package repoctx

import "testing"

func TestFallbackIndexesBoundedLexicalTermsWithoutASTRelations(t *testing.T) {
	cases := []struct{ path, src, term string }{
		{"scripts/order_worker.py", "def process_order(order_id): return order_id", "process_order"},
		{"web/order.ts", "export function createOrder(id: string) {}", "createorder"},
		{"db/order.sql", "select order_id from orders where status = 'new'", "order_id"},
		{"config/order.xml", "<bean id=\"orderService\" class=\"x.OrderService\"/>", "orderservice"},
	}
	for _, tc := range cases {
		f := ExtractFallback(tc.path, []byte(tc.src))
		if f.Path != tc.path {
			t.Fatalf("path=%q want=%q", f.Path, tc.path)
		}
		if len(f.Symbols) != 0 || len(f.Relations) != 0 || len(f.candidates) != 0 {
			t.Fatalf("fallback fabricated structure: %+v", f)
		}
		if !containsString(f.Terms, tc.term) {
			t.Fatalf("terms=%v missing %q", f.Terms, tc.term)
		}
	}
}

func TestFallbackTermsAreBoundedAndDeterministic(t *testing.T) {
	src := []byte("alpha beta gamma alpha delta epsilon zeta eta theta iota kappa lambda")
	a := ExtractFallback("x.py", src)
	b := ExtractFallback("x.py", src)
	if len(a.Terms) > fallbackMaxTerms {
		t.Fatalf("terms unbounded: %d", len(a.Terms))
	}
	if len(a.Terms) != len(b.Terms) {
		t.Fatal("fallback unstable")
	}
	for i := range a.Terms {
		if a.Terms[i] != b.Terms[i] {
			t.Fatalf("fallback unstable: %v vs %v", a.Terms, b.Terms)
		}
	}
}
