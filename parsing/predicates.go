package parsing

import (
	"fmt"
	"strings"

	"github.com/adam000/foreach-git-dir/predicate"
)

type predicateType int

func (t predicateType) ToString() string {
	switch t {
	case pNone:
		return "none"
	case pFlag:
		return "flag"
	case pOpenParen:
		return "("
	case pCloseParen:
		return ")"
	case pAnd:
		return "-and"
	case pOr:
		return "-or"
	case pNot:
		return "-not"
	}

	return "unknown type"
}

const (
	pNone predicateType = iota
	pFlag
	pOpenParen
	pCloseParen
	pAnd
	pOr
	pNot
)

type predicateToken struct {
	typ  predicateType
	flag string
	text string
}

func (p predicateToken) String() string {
	return fmt.Sprintf("(%s|%s|%s)", p.typ.ToString(), p.flag, p.text)
}

type predicateInfo struct {
	Name        string
	Description string
	Typ         predicateType
}

// This isn't used to its fullest here; this should also be used to print the usage string
func PredicateInfo() map[string]predicateInfo {
	return map[string]predicateInfo{
		"-and": {
			Name:        "-and",
			Description: "Combine two or more predicates where both must be true",
			Typ:         pAnd,
		},
		"-or": {
			Name:        "-or",
			Description: "Combine two or more predicates where either one must be true",
			Typ:         pOr,
		},
		"-not": {
			Name:        "-not",
			Description: "Negate the following predicate",
			Typ:         pNot,
		},
		"-isdirty": {
			Name:        "-isDirty",
			Description: "Is the repository dirty?",
			Typ:         pFlag,
		},
		"-custom": {
			Name:        "-custom",
			Description: "Make a custom predicate",
			Typ:         pFlag,
		},
	}
}

type predicateProvider struct {
	and     func(p1, p2 predicate.Predicate) predicate.Predicate
	or      func(p1, p2 predicate.Predicate) predicate.Predicate
	not     func(pred predicate.Predicate) predicate.Predicate
	custom  func(string) predicate.Predicate
	isDirty predicate.Predicate
}

var predProvider = predicateProvider{
	and:     predicate.And,
	or:      predicate.Or,
	not:     predicate.Not,
	custom:  predicate.Custom,
	isDirty: predicate.IsDirty,
}

func tokenizePredicates(args []string, argIndex int) ([]predicateToken, int, error) {
	numArgs := len(args)
	predicateDivider := "--"

	pTok := make([]predicateToken, 0)
	tokenMap := PredicateInfo()
	for numArgs != argIndex {
		thisArg := strings.Trim(strings.ToLower(args[argIndex]), " \t")
		if thisArg == predicateDivider {
			argIndex++
			break
		}

		// tokenize as many parens as possible at the beginning of thisArg
		for len(thisArg) != 0 && thisArg[0] == '(' {
			pTok = append(pTok, predicateToken{typ: pOpenParen})
			thisArg = thisArg[1:]
		}

		// Take off end parens, keep track of them
		numEndParens := 0
		for len(thisArg) != 0 && thisArg[len(thisArg)-1] == ')' {
			numEndParens++
			thisArg = thisArg[:len(thisArg)-1]
		}

		if len(thisArg) != 0 {
			// Tokenize the arguments and deal with it down the line
			if info, ok := tokenMap[thisArg]; ok {
				if info.Typ == pFlag && strings.ToLower(info.Name) == "-custom" {
					// Can't have end parens after -custom but before the script
					if numEndParens != 0 {
						return []predicateToken{}, argIndex, fmt.Errorf("can't have end parentheses immediately after -custom; argument required")
					}

					// consume the next token
					argIndex++
					customCmd := args[argIndex]

					// Take off end parens, keep track of them
					for len(customCmd) != 0 && customCmd[len(customCmd)-1] == ')' {
						numEndParens++
						customCmd = customCmd[:len(customCmd)-1]
					}
					if len(customCmd) == 0 {
						return []predicateToken{}, argIndex, fmt.Errorf("can't have end parentheses immediately after -custom; argument required")
					}

					pTok = append(pTok, predicateToken{
						typ:  pFlag,
						flag: strings.ToLower(info.Name),
						text: customCmd,
					})
				} else {
					pTok = append(pTok, predicateToken{
						typ:  info.Typ,
						flag: strings.ToLower(info.Name),
					})
				}
			} else {
				return []predicateToken{}, argIndex, fmt.Errorf("could not find predicate '%s' (did you forget to include '--' to separate predicates and actions?)", thisArg)
			}
		}

		// Add close parens after, as necessary
		for i := 0; i < numEndParens; i++ {
			pTok = append(pTok, predicateToken{typ: pCloseParen})
		}
		argIndex++
	}

	return pTok, argIndex, nil
}

type predicateParser struct {
	tokens       []predicateToken
	currentToken int
	provider     predicateProvider
}

func (p predicateParser) allTokensConsumed() bool {
	return len(p.tokens) == p.currentToken
}

func (p predicateParser) stoppingPoint() bool {
	return p.allTokensConsumed() || p.tokens[p.currentToken].typ == pCloseParen
}

const (
	customFlag  = "-custom"
	isDirtyFlag = "-isdirty"
)

func (p *predicateParser) parseFlag() (predicate.Predicate, error) {
	token := p.tokens[p.currentToken]
	switch token.flag {
	case customFlag:
		p.currentToken++
		return p.provider.custom(token.text), nil
	case isDirtyFlag:
		p.currentToken++
		return p.provider.isDirty, nil
	default:
		return predicate.Id, fmt.Errorf("unknown flag '%s'", token)
	}
}

func (p *predicateParser) parseSubExpressionAnd(left predicate.Predicate) (predicate.Predicate, error) {
	right, err := p.parseSubExpression()
	if err != nil {
		return left, err
	}

	pred, err := p.parseBinaryExpression(p.provider.and(left, right))
	return pred, err
}

// Parse when we expect and or or (or a close paren in the case that we only have a left)
func (p *predicateParser) parseBinaryExpression(left predicate.Predicate) (predicate.Predicate, error) {
	if p.stoppingPoint() {
		return left, nil
	}

	switch p.tokens[p.currentToken].typ {
	case pCloseParen:
		return left, nil
	case pAnd:
		p.currentToken++
		pred, err := p.parseSubExpressionAnd(left)
		return pred, err
	case pOr:
		p.currentToken++
		right, err := p.parseExpression()
		return p.provider.or(left, right), err
	default:
		return predicate.Id, fmt.Errorf("unexpected token %s, expected -and or -or", p.tokens[p.currentToken].typ.ToString())
	}
}

func (p *predicateParser) parseExpression() (predicate.Predicate, error) {
	if p.stoppingPoint() {
		return predicate.Id, nil
	}

	left, err := p.parseSubExpression()
	if err != nil || p.stoppingPoint() {
		return left, err
	}

	pred, err := p.parseBinaryExpression(left)
	return pred, err
}

// A sub-expression is an expression contained within parentheses, a -not followed by an expression, or
// just a flag.
func (p *predicateParser) parseSubExpression() (predicate.Predicate, error) {
	if p.allTokensConsumed() {
		return predicate.Id, fmt.Errorf("unexpected end of input")
	}
	switch p.tokens[p.currentToken].typ {
	case pFlag:
		left, err := p.parseFlag()
		return left, err
	case pNot:
		p.currentToken++
		pred, err := p.parseSubExpression()
		return p.provider.not(pred), err
	case pOpenParen:
		p.currentToken++
		if !p.allTokensConsumed() && p.tokens[p.currentToken].typ == pCloseParen {
			p.currentToken++
			// In this case, we have `()`. It might be due to shell expansion, so we're going to accept it.
			return predicate.Id, nil
		}
		pred, err := p.parseExpression()
		if err != nil {
			return pred, err
		}
		if p.allTokensConsumed() {
			return pred, fmt.Errorf("missing close paren")
		}
		p.currentToken++
		return pred, nil
	}
	return predicate.Id, fmt.Errorf("unexpected %s, was expecting a flag, '-not', or '('", p.tokens[p.currentToken].typ.ToString())
}

func parsePredicates(args []string, argIndex int) (predicate.Predicate, int, error) {
	tokens, argIndex, err := tokenizePredicates(args, argIndex)
	if err != nil {
		return predicate.Id, argIndex, fmt.Errorf("tokenizing predicates: %w", err)
	}

	if len(tokens) != 0 {
		p := predicateParser{
			tokens:   tokens,
			provider: predProvider,
		}

		pred, err := p.parseExpression()
		if err != nil {
			err = fmt.Errorf("parsing predicates: %w", err)
		}
		if err == nil && !p.allTokensConsumed() {
			err = fmt.Errorf("did not consume all tokens (%d/%d)", p.currentToken, len(p.tokens))
		}
		return pred, argIndex, err
	}

	return predicate.Id, argIndex, nil
}
