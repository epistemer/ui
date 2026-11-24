package rule

import (
	"context"
	"regexp/syntax"
	"strings"
	"testing"
)

func generateSample(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpLiteral:
		return string(re.Rune)
	case syntax.OpCharClass:
		if len(re.Rune) > 0 {
			return string(re.Rune[0])
		}
		return ""
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return "a"
	case syntax.OpBeginText, syntax.OpEndText, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpWordBoundary, syntax.OpNoWordBoundary, syntax.OpEmptyMatch:
		return ""
	case syntax.OpCapture:
		return generateSample(re.Sub[0])
	case syntax.OpConcat:
		var sb strings.Builder
		for _, sub := range re.Sub {
			sb.WriteString(generateSample(sub))
		}
		return sb.String()
	case syntax.OpAlternate:
		if len(re.Sub) > 0 {
			return generateSample(re.Sub[0])
		}
		return ""
	case syntax.OpStar, syntax.OpQuest:
		return ""
	case syntax.OpPlus:
		return generateSample(re.Sub[0])
	case syntax.OpRepeat:
		var sb strings.Builder
		s := generateSample(re.Sub[0])
		for i := 0; i < re.Min; i++ {
			sb.WriteString(s)
		}
		return sb.String()
	default:
		return ""
	}
}

func TestPostcodePatternSamples(t *testing.T) {
	for country, pattern := range postcodePatterns {
		t.Run(country, func(t *testing.T) {
			parsed, err := syntax.Parse(pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("syntax.Parse(%q): %v", country, err)
			}
			sample := generateSample(parsed)
			re := postcodeRegexes[country]
			if !re.MatchString(sample) {
				t.Errorf("%s: sample %q does not match pattern %s", country, sample, pattern)
			}
			if re.MatchString("junk " + sample + " junk") {
				t.Errorf("%s: pattern matched junk around sample %q", country, sample)
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
