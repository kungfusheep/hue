package query

import (
	"fmt"
	"strings"

	"github.com/kungfusheep/hue/client"
)

// ExprNode represents a node in the expression tree (internal AST)
type exprNode interface {
	evaluate(lights []client.Light, ctx *Context) []client.Light
}

// andNode represents AND operation
type andNode struct {
	children []exprNode
}

func (n *andNode) evaluate(lights []client.Light, ctx *Context) []client.Light {
	result := lights
	for _, child := range n.children {
		result = child.evaluate(result, ctx)
	}
	return result
}

// orNode represents OR operation
type orNode struct {
	children []exprNode
}

func (n *orNode) evaluate(lights []client.Light, ctx *Context) []client.Light {
	var result []client.Light
	seen := make(map[string]bool)

	for _, child := range n.children {
		matched := child.evaluate(lights, ctx)
		for _, light := range matched {
			if !seen[light.ID] {
				result = append(result, light)
				seen[light.ID] = true
			}
		}
	}

	return result
}

// notNode represents NOT operation
type notNode struct {
	child exprNode
}

func (n *notNode) evaluate(lights []client.Light, ctx *Context) []client.Light {
	matched := n.child.evaluate(lights, ctx)
	return exclude(lights, matched)
}

// filterNode represents a single filter (leaf node)
type filterNode struct {
	filter Filter
}

func (n *filterNode) evaluate(lights []client.Light, ctx *Context) []client.Light {
	return applyFilterWithContext(n.filter, lights, ctx)
}

// parseBrackets parses a query string with bracket support
// Returns an expression tree (AST)
func parseBrackets(queryStr string) (exprNode, error) {
	// Tokenize the query
	tokens, err := tokenize(queryStr)
	if err != nil {
		return nil, err
	}

	// Parse tokens into expression tree
	expr, _, err := parseExpression(tokens, 0)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

// token represents a parsed token
type token struct {
	typ   tokenType
	value string
}

type tokenType int

const (
	tokenFilter tokenType = iota
	tokenLParen
	tokenRParen
	tokenComma
)

// tokenize converts query string to tokens
func tokenize(s string) ([]token, error) {
	var tokens []token
	var current strings.Builder
	depth := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch ch {
		case '(':
			// Flush current token if any
			if current.Len() > 0 {
				tokens = append(tokens, token{typ: tokenFilter, value: strings.TrimSpace(current.String())})
				current.Reset()
			}
			tokens = append(tokens, token{typ: tokenLParen})
			depth++

		case ')':
			// Flush current token if any
			if current.Len() > 0 {
				tokens = append(tokens, token{typ: tokenFilter, value: strings.TrimSpace(current.String())})
				current.Reset()
			}
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unmatched closing parenthesis at position %d", i)
			}
			tokens = append(tokens, token{typ: tokenRParen})

		case ',':
			// Flush current token if any
			if current.Len() > 0 {
				tokens = append(tokens, token{typ: tokenFilter, value: strings.TrimSpace(current.String())})
				current.Reset()
			}
			tokens = append(tokens, token{typ: tokenComma})

		case ' ', '\t':
			// Space is a separator - flush if we have content
			if current.Len() > 0 {
				tokens = append(tokens, token{typ: tokenFilter, value: strings.TrimSpace(current.String())})
				current.Reset()
			}

		default:
			current.WriteByte(ch)
		}
	}

	// Flush final token
	if current.Len() > 0 {
		tokens = append(tokens, token{typ: tokenFilter, value: strings.TrimSpace(current.String())})
	}

	if depth != 0 {
		return nil, fmt.Errorf("unmatched parentheses: %d unclosed", depth)
	}

	return tokens, nil
}

// parseExpression recursively parses tokens into an expression tree
// Returns: (node, nextPosition, error)
func parseExpression(tokens []token, pos int) (exprNode, int, error) {
	if pos >= len(tokens) {
		return nil, pos, fmt.Errorf("unexpected end of expression")
	}

	// Collect terms at this level
	var terms []exprNode
	currentOp := "AND" // Default operator

	for pos < len(tokens) {
		tok := tokens[pos]

		switch tok.typ {
		case tokenLParen:
			// Check if previous token was a negation marker
			isNegated := false
			if len(terms) > 0 {
				// Check if we should look back for negation
				// This is handled by the tokenFilter case instead
			}

			// Recursively parse parenthesized expression
			subExpr, newPos, err := parseExpression(tokens, pos+1)
			if err != nil {
				return nil, pos, err
			}

			if isNegated {
				subExpr = &notNode{child: subExpr}
			}

			terms = append(terms, subExpr)
			pos = newPos

		case tokenRParen:
			// End of this group
			return buildNode(terms, currentOp), pos + 1, nil

		case tokenComma:
			// Switch to OR mode
			currentOp = "OR"
			pos++

		case tokenFilter:
			// Check for negation
			if strings.HasPrefix(tok.value, "-") {
				// Negation - check if next token is a paren
				if pos+1 < len(tokens) && tokens[pos+1].typ == tokenLParen {
					// Negation of bracketed expression: -(...)
					subExpr, newPos, err := parseExpression(tokens, pos+2)
					if err != nil {
						return nil, pos, err
					}
					terms = append(terms, &notNode{child: subExpr})
					pos = newPos
				} else {
					// Negation of simple filter
					filter := parseFilter(strings.TrimPrefix(tok.value, "-"))
					terms = append(terms, &notNode{child: &filterNode{filter: filter}})
					pos++
				}
			} else {
				// Normal filter
				filter := parseFilter(tok.value)
				terms = append(terms, &filterNode{filter: filter})
				pos++
			}

		default:
			pos++
		}

		// If we hit end without closing paren and we're at top level, break
		if pos >= len(tokens) {
			break
		}
	}

	return buildNode(terms, currentOp), pos, nil
}

// buildNode combines terms with the specified operator
func buildNode(terms []exprNode, op string) exprNode {
	if len(terms) == 0 {
		return &andNode{children: []exprNode{}}
	}
	if len(terms) == 1 {
		return terms[0]
	}

	if op == "OR" {
		return &orNode{children: terms}
	}
	return &andNode{children: terms}
}

// hasBrackets checks if a query contains parentheses
func hasBrackets(s string) bool {
	return strings.Contains(s, "(") || strings.Contains(s, ")")
}
