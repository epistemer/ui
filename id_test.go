package ui_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/kerneltest"
)

func seq(prefix string) context.Context {
	return ui.WithIDs(context.Background(), ui.Sequence(prefix))
}

func TestScopeWiresElements(t *testing.T) {
	got := render(t, seq("t"), kerneltest.Field())
	want := `<label for="email--t-1">Email</label> <input id="email--t-1" aria-describedby="email--t-1-hint"> <small id="email--t-1-hint">We never share it.</small>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestScopeNesting(t *testing.T) {
	if got, want := render(t, seq("t"), kerneltest.NestedScopes()), "<a>email--t-1</a><b>email--t-2</b><c>email--t-1</c>"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestScopeSiblings(t *testing.T) {
	if got, want := render(t, seq("t"), kerneltest.Siblings()), "<a>email--t-1</a><b>email--t-2</b><c>email--t-3 hint--t-4</c>"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestGetOutsideScopePanics(t *testing.T) {
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "email.Get outside ui.Scope(email)") {
			t.Errorf("panic %v, want one naming the id and the fix", r)
		}
	}()
	kerneltest.GetOutsideScope().Render(context.Background(), io.Discard)
}

func TestScopeErrors(t *testing.T) {
	if err := kerneltest.Twice().Render(context.Background(), io.Discard); err == nil || !strings.Contains(err.Error(), `"email" twice`) {
		t.Errorf("an id listed twice: got %v", err)
	}
	for _, tok := range []string{"", "A", "a--b", "a b", "-a", "a-", "a_b"} {
		ctx := ui.WithIDs(context.Background(), fixed(tok))
		err := kerneltest.Field().Render(ctx, io.Discard)
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", tok)) || !strings.Contains(err.Error(), "ui_test.fixed") {
			t.Errorf("token %q: got %v, want an error naming the token and the source", tok, err)
		}
	}
	for _, tok := range []string{"1-2", "a", "k3x9a2-1f"} {
		if err := kerneltest.Field().Render(ui.WithIDs(context.Background(), fixed(tok)), io.Discard); err != nil {
			t.Errorf("token %q: %v", tok, err)
		}
	}
	drawn := ui.Sequence("t")
	ui.Scope(kerneltest.Email, kerneltest.Hint, kerneltest.Email).Render(ui.WithIDs(context.Background(), drawn), io.Discard)
	if next := drawn.Next(); next != "t-1" {
		t.Errorf("a Scope listing an id twice drew tokens: next is %q", next)
	}
}

type fixed string

func (f fixed) Next() string { return string(f) }

func TestDeclarationPanics(t *testing.T) {
	for name, f := range map[string]func(){
		`NewID("Email")`:   func() { ui.NewID("Email") },
		`NewID("a--b")`:    func() { ui.NewID("a--b") },
		`NewID("1a")`:      func() { ui.NewID("1a") },
		`NewID("")`:        func() { ui.NewID("") },
		`Sequence("")`:     func() { ui.Sequence("") },
		`Sequence("a b")`:  func() { ui.Sequence("a b") },
		"WithIDs nil":      func() { ui.WithIDs(context.Background(), nil) },
		"zero ID Key":      func() { ui.ID{}.Key("x") },
		"zero ID Get":      func() { ui.ID{}.Get(context.Background()) },
		"zero ID Part":     func() { ui.ID{}.Part(context.Background(), "x") },
		"zero ID in Scope": func() { ui.Scope(ui.ID{}) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			f()
		})
	}
}

func TestKey(t *testing.T) {
	row := ui.NewID("order-row")
	for _, tc := range []struct {
		parts []string
		want  string
	}{
		{nil, "order-row"},
		{[]string{"1042"}, "order-row-1042"},
		{[]string{"A/7"}, "order-row-A_2f7"},
		{[]string{""}, "order-row-_"},
		{[]string{"a-b"}, "order-row-a_2db"},
		{[]string{"x", "y"}, "order-row-x-y"},
		{[]string{"é"}, "order-row-_c3_a9"},
	} {
		if got := row.Key(tc.parts...); got != tc.want {
			t.Errorf("Key(%q) = %q, want %q", tc.parts, got, tc.want)
		}
	}
}

var tokenRE = regexp.MustCompile(`^email--[0-9a-z]{6}-[0-9a-z]+$`)

func TestProcessSource(t *testing.T) {
	const goroutines, each = 64, 10_000
	ids := make([][]string, goroutines)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := make([]string, 0, each)
			c := ui.Scope(kerneltest.Email)
			grab := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
				out = append(out, kerneltest.Email.Get(ctx))
				return nil
			})
			ctx := templ.WithChildren(context.Background(), grab)
			for range each {
				if err := c.Render(ctx, io.Discard); err != nil {
					t.Error(err)
					return
				}
			}
			ids[g] = out
		}()
	}
	wg.Wait()
	seen := make(map[string]bool, goroutines*each)
	for _, out := range ids {
		for _, id := range out {
			if seen[id] {
				t.Fatalf("duplicate id %q", id)
			}
			if !tokenRE.MatchString(id) {
				t.Fatalf("id %q is not <name>--<6 base-36>-<base-36 counter>", id)
			}
			seen[id] = true
		}
	}
	if len(seen) != goroutines*each {
		t.Errorf("%d ids, want %d", len(seen), goroutines*each)
	}
}

func TestSequenceIsByteStable(t *testing.T) {
	a := render(t, seq("t"), kerneltest.Siblings())
	b := render(t, seq("t"), kerneltest.Siblings())
	if a != b {
		t.Errorf("two renders with Sequence differ:\n%s\n%s", a, b)
	}
}

func TestNewIDPanicNamesTheValue(t *testing.T) {
	defer func() {
		if msg, _ := recover().(string); !strings.Contains(msg, `"Bad"`) {
			t.Errorf("panic %q does not name the value", msg)
		}
	}()
	ui.NewID("Bad")
}

// TestHandlesWithOneName pins K8: two handles named alike never meet.
func TestHandlesWithOneName(t *testing.T) {
	a, b := ui.NewID("email"), ui.NewID("email")
	var got []string
	grab := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
		got = append(got, a.Get(ctx), b.Get(ctx))
		return nil
	})
	if err := ui.Scope(a, b).Render(templ.WithChildren(seq("t"), grab), io.Discard); err != nil {
		t.Fatalf("Scope(a, b) of two handles named alike: %v", err)
	}
	if got[0] == got[1] {
		t.Errorf("both handles got %q", got[0])
	}
	defer func() {
		if recover() == nil {
			t.Error("b.Get inside Scope(a) did not panic")
		}
	}()
	only := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error { b.Get(ctx); return nil })
	ui.Scope(a).Render(templ.WithChildren(seq("t"), only), io.Discard)
}

// TestScopeClearsChildren pins K3 for Scope.
func TestScopeClearsChildren(t *testing.T) {
	if got := render(t, seq("t"), kerneltest.ChildInScope()); got != "<i></i>" {
		t.Errorf("got %q, want %q", got, "<i></i>")
	}
}

func TestSequenceConcurrent(t *testing.T) {
	s := ui.Sequence("t")
	var mu sync.Mutex
	seen := map[string]bool{}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				tok := s.Next()
				mu.Lock()
				if seen[tok] {
					t.Errorf("duplicate token %q", tok)
				}
				seen[tok] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(seen) != 16000 || !seen["t-16000"] {
		t.Errorf("%d tokens; want t-1 … t-16000", len(seen))
	}
}
