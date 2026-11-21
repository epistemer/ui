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
	for _, tok := range []string{"", "A", "a--b", "a b", "-a", "a-"} {
		ctx := ui.WithIDs(context.Background(), fixed(tok))
		err := kerneltest.Field().Render(ctx, io.Discard)
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%q", tok)) {
			t.Errorf("token %q: got %v, want an error naming it", tok, err)
		}
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
