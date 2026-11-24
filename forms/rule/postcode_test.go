package rule

import (
	"context"
	"regexp/syntax"
	"testing"
)

// samples gives short strings re matches: every branch of an
// alternation, and an optional part both absent and present, so a test
// reaches each alternative of a pattern (the anchoring bug of GB and CR
// lived in a later one). It stops growing at limit strings.
func samples(re *syntax.Regexp, limit int) []string {
	cross := func(a, b []string) []string {
		var out []string
		for _, x := range a {
			for _, y := range b {
				if len(out) == limit {
					return out
				}
				out = append(out, x+y)
			}
		}
		return out
	}
	switch re.Op {
	case syntax.OpLiteral:
		return []string{string(re.Rune)}
	case syntax.OpCharClass:
		if len(re.Rune) == 0 {
			return nil
		}
		return []string{string(re.Rune[0])}
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return []string{"A"}
	case syntax.OpCapture:
		return samples(re.Sub[0], limit)
	case syntax.OpConcat:
		out := []string{""}
		for _, sub := range re.Sub {
			out = cross(out, samples(sub, limit))
		}
		return out
	case syntax.OpAlternate:
		var out []string
		for _, sub := range re.Sub {
			for _, x := range samples(sub, limit) {
				if len(out) < limit {
					out = append(out, x)
				}
			}
		}
		return out
	case syntax.OpQuest, syntax.OpStar:
		return append([]string{""}, samples(re.Sub[0], limit)...)
	case syntax.OpPlus:
		return samples(re.Sub[0], limit)
	case syntax.OpRepeat:
		out := []string{""}
		for range re.Min {
			out = cross(out, samples(re.Sub[0], limit))
		}
		if re.Min == 0 {
			out = append(out, samples(re.Sub[0], limit)...)
		}
		return out
	default: // anchors, empty match, word boundaries
		return []string{""}
	}
}

func TestPostcodePatternSamples(t *testing.T) {
	for country, pattern := range postcodePatterns {
		t.Run(country, func(t *testing.T) {
			parsed, err := syntax.Parse(pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("syntax.Parse(%q): %v", country, err)
			}
			re := postcodeRegexes[country]
			got := samples(parsed, 64)
			if len(got) == 0 {
				t.Fatalf("%s: no sample of %s", country, pattern)
			}
			for _, sample := range got {
				if !re.MatchString(sample) {
					t.Errorf("%s: sample %q does not match pattern %s", country, sample, pattern)
				}
				// Junk on either side: an alternative anchored at one end
				// only (CR's) passes junk on the other.
				for _, junk := range []string{"junk " + sample + " junk", "junk " + sample, sample + " junk"} {
					if re.MatchString(junk) {
						t.Errorf("%s: pattern matched %q", country, junk)
					}
				}
			}
		})
	}
}

func TestPostCodeTable(t *testing.T) {
	ctx := context.Background()

	// GB "SW1A 1AA" and "sw1a 1aa" pass, "junk SW1A 1AA junk" fails;
	gb := PostCode("GB")
	if err := Check(ctx, "SW1A 1AA", gb); err != nil {
		t.Errorf("GB SW1A 1AA: %v", err)
	}
	if err := Check(ctx, "sw1a 1aa", gb); err != nil {
		t.Errorf("GB sw1a 1aa: %v", err)
	}
	if err := Check(ctx, "junk SW1A 1AA junk", gb); err == nil {
		t.Error("GB junk SW1A 1AA junk should fail")
	}

	// PostCode("gb") behaves as "GB";
	gbLower := PostCode("gb")
	if err := Check(ctx, "SW1A 1AA", gbLower); err != nil {
		t.Errorf("gb SW1A 1AA: %v", err)
	}
	if err := Check(ctx, "sw1a 1aa", gbLower); err != nil {
		t.Errorf("gb sw1a 1aa: %v", err)
	}
	if err := Check(ctx, "junk SW1A 1AA junk", gbLower); err == nil {
		t.Error("gb junk SW1A 1AA junk should fail")
	}

	// PostCode("ZZ") and PostCode("") pass "anything";
	zz := PostCode("ZZ")
	if err := Check(ctx, "anything", zz); err != nil {
		t.Errorf("ZZ anything: %v", err)
	}
	empty := PostCode("")
	if err := Check(ctx, "anything", empty); err != nil {
		t.Errorf("empty anything: %v", err)
	}

	// HasPostCode("us"), HasPostCode(" GB ") true, HasPostCode("ZZ") false.
	if !HasPostCode("us") {
		t.Error(`HasPostCode("us") should be true`)
	}
	if !HasPostCode(" GB ") {
		t.Error(`HasPostCode(" GB ") should be true`)
	}
	if HasPostCode("ZZ") {
		t.Error(`HasPostCode("ZZ") should be false`)
	}
}
