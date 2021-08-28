package parsing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/adam000/foreach-git-dir/predicate"
)

func TestNoPredicateTokenization(t *testing.T) {
	input := []string{"--", "-PrintBriefStatus"}
	argIndex := 0

	tokens, argIndex, err := tokenizePredicates(input, argIndex)

	if err != nil {
		t.Errorf("Got error testing without predicates: %v", err)
	}

	if len(tokens) != 0 {
		t.Errorf("Got tokens when there shouldn't have been any: %#v", tokens)
	}

	if argIndex != 1 {
		t.Errorf("Expected argIndex to advance to 1, it was %d", argIndex)
	}
}

func TestSinglePredicatesTokenization(t *testing.T) {
	inputs := [][]string{
		[]string{"-IsDirty", "--", "-PrintBriefStatus"},
		[]string{"-isdirty", "--", "-PrintBriefStatus"},
		[]string{"-And", "--", "-PrintBriefStatus"},
		[]string{"-Or", "--", "-PrintBriefStatus"},
		[]string{"-Not", "--", "-PrintBriefStatus"},
	}
	argIndex := 0

	for _, input := range inputs {
		tokens, argIndex, err := tokenizePredicates(input, argIndex)

		if err != nil {
			t.Errorf("Got error testing one predicate: %v", err)
		}

		if len(tokens) != 1 {
			t.Errorf("Got %d tokens when there should have been %d: %#v", len(tokens), 1, tokens)
		}

		if argIndex != 2 {
			t.Errorf("Expected argIndex to advance to 2, it was %d", argIndex)
		}
	}
}

func TestValidParenTokenization(t *testing.T) {
	inputs := [][]string{
		[]string{"(-IsDirty)", "--", "-PrintBriefStatus"},
		[]string{"(", "-IsDirty", ")", "--", "-PrintBriefStatus"},
		[]string{"(-IsDirty", ")", "--", "-PrintBriefStatus"},
		[]string{"(", "-IsDirty)", "--", "-PrintBriefStatus"},
	}
	argIndex := 0

	for _, input := range inputs {
		tokens, _, err := tokenizePredicates(input, argIndex)

		if err != nil {
			t.Errorf("Got error testing parens: %v", err)
		}

		if len(tokens) != 3 {
			t.Errorf("Got %d tokens when there should have been %d: %#v", len(tokens), 3, tokens)
		}

		if tokens[0].typ != pOpenParen {
			t.Errorf("Token 0 should have been %d, was %d", pOpenParen, tokens[0].typ)
		}
		if tokens[1].typ != pFlag {
			t.Errorf("Token 1 should have been %d, was %d", pFlag, tokens[1].typ)
		}
		if tokens[2].typ != pCloseParen {
			t.Errorf("Token 2 should have been %d, was %d", pCloseParen, tokens[2].typ)
		}
	}
}

func TestInvalidParenTokenization(t *testing.T) {
	inputs := [][]string{
		[]string{"(-Custom)", "--", "-PrintBriefStatus"},
		[]string{"(-Custom", ")", "--", "-PrintBriefStatus"},
	}
	argIndex := 0

	for _, input := range inputs {
		_, _, err := tokenizePredicates(input, argIndex)

		if err == nil {
			t.Errorf("Didn't get error testing invalid parens")
		}
	}
}

// Test when argIndex != 0
func TestNonZeroArgIndexTokenization(t *testing.T) {
	cases := []struct {
		input    []string
		argIndex int
	}{
		{
			[]string{"foobar", "-IsDirty", "--", "-PrintBriefStatus"},
			1,
		},
		{
			[]string{"foobar", "baz", "-IsDirty", "--", "-PrintBriefStatus"},
			2,
		},
	}

	for _, testCase := range cases {
		input := testCase.input
		argIndex := testCase.argIndex

		tokens, _, err := tokenizePredicates(input, argIndex)

		if err != nil {
			t.Errorf("Got error testing argIndex: %v", err)
		}

		if len(tokens) != 1 {
			t.Errorf("Got %d tokens when there should have been %d: %#v", len(tokens), 1, tokens)
		}
	}
}

func TestCustomTokenization(t *testing.T) {
	inputs := [][]string{
		[]string{"-Custom", "\"asdf\"", "--", "-PrintBriefStatus"},
	}
	argIndex := 0

	for _, input := range inputs {
		tokens, _, err := tokenizePredicates(input, argIndex)

		if err != nil {
			t.Errorf("Got error testing custom predicate: %v", err)
		}

		if len(tokens) != 1 {
			t.Errorf("Got %d tokens when there should have been %d: %#v", len(tokens), 1, tokens)
		}

		if input[1] != tokens[0].text {
			t.Errorf("Token text should be %s, got %s", input[1], tokens[0].text)
		}
	}
}

func TestInvalidTokenization(t *testing.T) {
	inputs := [][]string{
		[]string{"-asdf", "--", "-PrintBriefStatus"},
		[]string{"-isDirty"},
	}
	argIndex := 0

	for _, input := range inputs {
		_, _, err := tokenizePredicates(input, argIndex)

		if err == nil {
			t.Errorf("Expected error testing invalid flag: %s", input[0])
		}
	}
}

// try parsing predicates
var testPredicateProvider = predicateProvider{
	and: predicate.And,
	or:  predicate.Or,
	not: predicate.Not,
	isDirty: func(root string) (bool, error) {
		return strings.Contains(root, "dirty"), nil
	},
	custom: func(command string) predicate.Predicate {
		return func(root string) (bool, error) {
			succeeds := strings.Contains(root, command)
			if strings.Contains(root, "fail") {
				return succeeds, fmt.Errorf("Custom failed to run")
			}
			return succeeds, nil
		}
	},
}

func TestEmptyParsing(t *testing.T) {
	test := []predicateToken{}

	p := parser{
		tokens:       test,
		currentToken: 0,
		provider:     testPredicateProvider,
	}
	_, err := p.parseExpression()

	if err != nil {
		t.Errorf("Got error testing empty predicate parsing: %v", err)
	}
}

func TestSimpleParsing(t *testing.T) {
	testCases := [][]predicateToken{
		[]predicateToken{predicateToken{typ: pFlag, flag: "-isdirty"}},
		[]predicateToken{
			predicateToken{typ: pOpenParen},
			predicateToken{typ: pFlag, flag: "-isdirty"},
			predicateToken{typ: pCloseParen},
		},
		[]predicateToken{
			predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
		},
	}

	for _, test := range testCases {
		p := parser{
			tokens:       test,
			currentToken: 0,
			provider:     testPredicateProvider,
		}
		pred, err := p.parseExpression()

		if err != nil {
			t.Fatalf("Got error testing simple predicate parsing: %v", err)
		}
		if pred == nil {
			t.Fatalf("nil predicate :(")
		}

		result, err := pred("dirty;asdf")
		if err != nil {
			t.Fatalf("Got error running simple predicate: %v", err)
		}

		if !result {
			t.Errorf("Expected simple predicate to return true, returned false")
		}
	}
}

func TestSimpleBinaryParsing(t *testing.T) {
	testCases := [][]predicateToken{
		[]predicateToken{
			predicateToken{typ: pFlag, flag: "-isdirty"},
			predicateToken{typ: pOr},
			predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
		},
		[]predicateToken{
			predicateToken{typ: pFlag, flag: "-isdirty"},
			predicateToken{typ: pAnd},
			predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
		},
		[]predicateToken{
			predicateToken{typ: pFlag, flag: "-isdirty"},
			predicateToken{typ: pOr},
			predicateToken{typ: pFlag, flag: "-custom", text: "foo"},
			predicateToken{typ: pOr},
			predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
		},
		[]predicateToken{
			predicateToken{typ: pFlag, flag: "-isdirty"},
			predicateToken{typ: pAnd},
			predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
			predicateToken{typ: pAnd},
			predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
		},
	}

	for _, test := range testCases {
		p := parser{
			tokens:       test,
			currentToken: 0,
			provider:     testPredicateProvider,
		}
		pred, err := p.parseExpression()

		if err != nil {
			t.Fatalf("Got error testing simple predicate parsing: %v", err)
		}
		if pred == nil {
			t.Fatalf("nil predicate :(")
		}

		result, err := pred("dirty;asdf;zxcv")
		if err != nil {
			t.Fatalf("Got error running simple predicate: %v", err)
		}

		if !result {
			t.Errorf("Expected simple predicate to return true, returned false")
		}
	}
}

func TestSimplePredCanFail(t *testing.T) {
	testCases := [][]predicateToken{
		[]predicateToken{predicateToken{typ: pFlag, flag: "-isdirty"}},
	}

	for _, test := range testCases {
		p := parser{
			tokens:       test,
			currentToken: 0,
			provider:     testPredicateProvider,
		}
		pred, err := p.parseExpression()

		if err != nil {
			t.Errorf("Got error testing simple predicate parsing: %v", err)
		}

		result, err := pred("")
		if err != nil {
			t.Errorf("Got error running simple predicate: %v", err)
		}

		if result {
			t.Errorf("Expected simple predicate to return false, returned true")
		}
	}
}

func TestPredicatePrecedence(t *testing.T) {
	testCases := []struct {
		pred1         []predicateToken
		pred2         []predicateToken
		pred1Desc     string
		pred2Desc     string
		shouldBeEqual bool
	}{
		// And has higher precedence than or
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred1Desc:     "asdf or zxcv and qwer",
			pred2Desc:     "asdf or (zxcv and qwer)",
			shouldBeEqual: true,
		},
		// or does not have the same / higher precedence than and
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr, flag: "-or"},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd, flag: "-and"},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen, text: "("},
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr, flag: "-or"},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pCloseParen, text: ")"},
				predicateToken{typ: pAnd, flag: "-and"},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "asdf or zxcv and qwer",
			pred2Desc:     "(asdf or zxcv) and qwer",
			shouldBeEqual: false,
		},
		// or is commutative (1)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pCloseParen},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "asdf or zxcv or qwer",
			pred2Desc:     "(asdf or zxcv) or qwer",
			shouldBeEqual: true,
		},
		// or is commutative (2)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred1Desc:     "asdf or zxcv or qwer",
			pred2Desc:     "asdf or (zxcv or qwer)",
			shouldBeEqual: true,
		},
		// and is commutative (1)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pCloseParen},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "asdf and zxcv and qwer",
			pred2Desc:     "(asdf and zxcv) and qwer",
			shouldBeEqual: true,
		},
		// and is commutative (2)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred1Desc:     "asdf and zxcv and qwer",
			pred2Desc:     "asdf and (zxcv and qwer)",
			shouldBeEqual: true,
		},
		// useless parens are useless (1)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred1Desc:     "asdf and zxcv and qwer",
			pred2Desc:     "(asdf and zxcv and qwer)",
			shouldBeEqual: true,
		},
		// useless parens are useless (2)
		{
			pred1: []predicateToken{
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "asdf"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred1Desc:     "asdf or zxcv or qwer",
			pred2Desc:     "(asdf or zxcv or qwer)",
			shouldBeEqual: true,
		},
		// Not has higher precedence than and
		{
			pred1: []predicateToken{
				predicateToken{typ: pNot},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pNot},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pCloseParen},
				predicateToken{typ: pAnd},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "not zxcv and qwer",
			pred2Desc:     "(not zxcv) and qwer",
			shouldBeEqual: true,
		},
		// Not has higher precedence than or
		{
			pred1: []predicateToken{
				predicateToken{typ: pNot},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred2: []predicateToken{
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pNot},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pCloseParen},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "not zxcv or qwer",
			pred2Desc:     "(not zxcv) or qwer",
			shouldBeEqual: true,
		},
		// Not has lower precedence than parens
		{
			pred1: []predicateToken{
				predicateToken{typ: pNot},
				predicateToken{typ: pOpenParen},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
				predicateToken{typ: pCloseParen},
			},
			pred2: []predicateToken{
				predicateToken{typ: pNot},
				predicateToken{typ: pFlag, flag: "-custom", text: "zxcv"},
				predicateToken{typ: pOr},
				predicateToken{typ: pFlag, flag: "-custom", text: "qwer"},
			},
			pred1Desc:     "not (zxcv or qwer)",
			pred2Desc:     "not zxcv or qwer",
			shouldBeEqual: false,
		},
	}

	for _, test := range testCases {
		p1 := parser{
			tokens:       test.pred1,
			currentToken: 0,
			provider:     testPredicateProvider,
		}
		pred1, err := p1.parseExpression()
		if err != nil {
			t.Fatalf("Unexpected error parsing predicate %s: %v", test.pred1Desc, err)
		}

		p2 := parser{
			tokens:       test.pred2,
			currentToken: 0,
			provider:     testPredicateProvider,
		}
		pred2, err := p2.parseExpression()
		if err != nil {
			t.Fatalf("Unexpected error parsing predicate %s: %v", test.pred2Desc, err)
		}

		// Build up all the strings
		allEqual := true
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				for k := 0; k < 2; k++ {
					asdf := ""
					if i == 0 {
						asdf = "asdf"
					}
					zxcv := ""
					if j == 0 {
						zxcv = "zxcv"
					}
					qwer := ""
					if k == 0 {
						qwer = "qwer"
					}

					customString := fmt.Sprintf("%s%s%s", asdf, zxcv, qwer)

					result1, err := pred1(customString)
					if err != nil {
						t.Fatalf("Unexpected error in predicate %s: %v", test.pred1Desc, err)
					}
					result2, err := pred2(customString)
					if err != nil {
						t.Fatalf("Unexpected error in predicate %s: %v", test.pred2Desc, err)
					}

					if test.shouldBeEqual {
						if result1 != result2 {
							t.Errorf("Predicate '%s' (%t) didn't match predicate '%s' (%t) for custom match '%s'", test.pred1Desc, result1, test.pred2Desc, result2, customString)
						}
					} else {
						if result1 != result2 {
							allEqual = false
						}
					}
				}
			}
		}
		if !test.shouldBeEqual && allEqual {
			t.Errorf("For predicates %s and %s, they shouldn't all be equal, but they were for all custom strings", test.pred1Desc, test.pred2Desc)
		}
	}
}

func TestParseFailures(t *testing.T) {
	testCases := []string{
		"-not -or -- -Asdf",
		"-not -and -- -Asdf",
		"-isDirty -not -isDirty -- -Asdf",
		"-isDirty -and -not -- -Asdf",
		"-isDirty -and -or -isDirty -- -Asdf",
		"( -isDirty -and -isDirty -- -Asdf",
		" -isDirty ) -and -isDirty -- -Asdf",
		" -isDirty  -and -isDirty ( -- -Asdf",
		")  -isDirty  -and -isDirty  -- -Asdf",
		" ( -isDirty  -and -isDirty -isDirty ) -- -Asdf",
	}

	for _, test := range testCases {
		args := strings.Fields(test)
		_, _, err := parsePredicates(args, 0)
		if err == nil {
			t.Errorf("Expected error parsing predicate, didn't get one")
		}
	}
}

func TestParseOther(t *testing.T) {
	testCases := []string{
		" (-isDirty ) -and -isDirty -- -PrintBriefStatus",
		" ((   ((-isDirty )) ) -and -isDirty ) -- -PrintBriefStatus",
		" () -and (-isDirty -and -isDirty ) -- -PrintBriefStatus",
	}

	for _, test := range testCases {
		args := strings.Fields(test)
		_, _, err := parsePredicates(args, 0)
		if err != nil {
			t.Errorf("Error parsing: %s", err)
		}
	}
}
