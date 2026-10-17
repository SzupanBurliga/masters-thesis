package graphql

import (
	"fmt"
	"os"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func loadSchema(t *testing.T) *ast.Schema {
	t.Helper()
	src, err := os.ReadFile("../../api/schema.graphql")
	if err != nil {
		t.Fatal(err)
	}
	s, gerr := gqlparser.LoadSchema(&ast.Source{Name: "s", Input: string(src)})
	if gerr != nil {
		t.Fatal(gerr)
	}
	return s
}

func TestLimits(t *testing.T) {
	schema := loadSchema(t)
	l := Limits{MaxDepth: 6, MaxCost: 400, MaxAliases: 10}
	measure := func(src string) (depth, cost, aliases int) {
		qd, errs := gqlparser.LoadQuery(schema, src)
		if len(errs) > 0 {
			t.Fatal(errs)
		}
		op := qd.Operations[0]
		depth = l.depth(op.SelectionSet, qd.Fragments, 0, &aliases)
		cost = l.cost(op.SelectionSet, qd.Fragments, nil, 0)
		return
	}
	d, c, _ := measure(`{ user(id:1){ id name orders(limit:10){ id items{ quantity product{ id name price } } } } }`)
	if d > 6 || c > 400 {
		t.Errorf("O3 powinno przechodzić: depth=%d cost=%d", d, c)
	}
	d, _, _ = measure(`{ user(id:1){ orders{ items{ product{ category{ parent{ parent{ id } } } } } } } }`)
	if d <= 6 {
		t.Errorf("głębokie zapytanie powinno przekraczać limit: %d", d)
	}
	_, c, _ = measure(`{ orders(limit:1000){ id user{ orders(limit:50){ id } } } }`)
	if c <= 400 {
		t.Errorf("kosztowne zapytanie powinno przekraczać limit: %d", c)
	}
	_, _, a := measure(`{ a:user(id:1){id} b:user(id:2){id} }`)
	if a != 2 {
		t.Errorf("aliasy: %d", a)
	}
	// limit 0 i ujemny mają być liczone jak wartość domyślna, a nie jak 0
	_, c0, _ := measure(`{ user(id:1){ orders(limit:0){ items{ product{ id } } } } }`)
	_, c10, _ := measure(`{ user(id:1){ orders(limit:10){ items{ product{ id } } } } }`)
	_, cn, _ := measure(`{ user(id:1){ orders(limit:-5){ items{ product{ id } } } } }`)
	if c0 != c10 || cn != c10 {
		t.Errorf("limit<1 powinien dawać koszt domyślny: 0=%d -5=%d 10=%d", c0, cn, c10)
	}
	// limit powyżej MaxLimit jest ucinany po stronie serwera
	_, c50, _ := measure(`{ user(id:1){ orders(limit:50){ id } } }`)
	_, c1000, _ := measure(`{ user(id:1){ orders(limit:1000){ id } } }`)
	if c50 != c1000 {
		t.Errorf("limit>50 powinien być ucięty: 50=%d 1000=%d", c50, c1000)
	}
}

// TestCalibration pilnuje, że próg MAX_COST (400) rozdziela ruch legalny od ataków z serii D.
func TestCalibration(t *testing.T) {
	schema := loadSchema(t)
	l := Limits{MaxDepth: 6, MaxCost: 400, MaxAliases: 10}
	cost := func(src string) int {
		qd, errs := gqlparser.LoadQuery(schema, src)
		if len(errs) > 0 {
			t.Fatal(errs)
		}
		return l.cost(qd.Operations[0].SelectionSet, qd.Fragments, nil, 0)
	}
	limitQ := func(n int) string {
		return fmt.Sprintf(`{ user(id:1){ orders(limit:%d){ id user{ orders(limit:%d){ id items{ quantity product{ id name } } } } } } }`, n, n)
	}
	aliasQ := func(n int) string {
		q := "{"
		for i := 0; i < n; i++ {
			q += fmt.Sprintf(" a%d: user(id:1){ orders(limit:10){ id items{ product{ name } } } }", i)
		}
		return q + " }"
	}
	o3 := cost(`{ user(id:1){ id name orders(limit:10){ id items{ quantity product{ id name price } } } } }`)
	if o3*10 > l.MaxCost {
		t.Errorf("O3 (%d) powinno mieć co najmniej 10-krotny zapas względem %d", o3, l.MaxCost)
	}
	for _, c := range []struct {
		name string
		got  int
		pass bool
	}{
		{"limit 10", cost(limitQ(10)), true},
		{"limit 25", cost(limitQ(25)), false},
		{"limit 50", cost(limitQ(50)), false},
		{"limit 1000 (ucięty do 50)", cost(limitQ(1000)), false},
		{"alias 5", cost(aliasQ(5)), true},
		{"alias 10", cost(aliasQ(10)), true}, // 10 aliasów mieści się w limicie aliasów i kosztu
	} {
		if (c.got <= l.MaxCost) != c.pass {
			t.Errorf("%s: koszt %d, oczekiwano pass=%v przy MaxCost=%d", c.name, c.got, c.pass, l.MaxCost)
		}
		t.Logf("%s: koszt %d", c.name, c.got)
	}
}
