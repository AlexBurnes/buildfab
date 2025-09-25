package buildfab

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlexBurnes/version-go/pkg/version"
)

// ExpressionContext provides context for evaluating expressions
type ExpressionContext struct {
	Variables map[string]string
	Inputs    map[string]string
	Matrix    map[string]string
	CI        bool
	Branch    string
}

// ExpressionResult represents the result of an expression evaluation
type ExpressionResult struct {
	Value interface{}
	Type  string // "bool", "string", "number"
}

// NewExpressionContext creates a new expression context with default values
func NewExpressionContext(variables map[string]string) *ExpressionContext {
	ctx := &ExpressionContext{
		Variables: make(map[string]string),
		Inputs:    make(map[string]string),
		Matrix:    make(map[string]string),
		CI:        false,
		Branch:    "",
	}

	// Add platform variables first
	platformVars := GetPlatformVariablesMap()
	for k, v := range platformVars {
		ctx.Variables[k] = v
	}

	// Copy user variables (these can override platform variables)
	for k, v := range variables {
		ctx.Variables[k] = v
	}

	// Set default values
	if os.Getenv("CI") != "" {
		ctx.CI = true
	}

	// Try to get branch from git
	if branch, err := getGitBranch(); err == nil {
		ctx.Branch = branch
	}

	return ctx
}

// EvaluateExpression evaluates a when condition expression
func EvaluateExpression(expr string, ctx *ExpressionContext) (bool, error) {
	// Remove ${{ }} wrapper if present
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "${{") && strings.HasSuffix(expr, "}}") {
		expr = strings.TrimSpace(expr[3 : len(expr)-2])
	}

	// Parse and evaluate the expression
	result, err := parseExpression(expr, ctx)
	if err != nil {
		return false, err
	}

	// Convert result to boolean
	return toBool(result), nil
}

// ParseExpression parses an expression and returns the result (for debugging)
func ParseExpression(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	return parseExpression(expr, ctx)
}

// parseExpression parses and evaluates an expression
func parseExpression(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	expr = strings.TrimSpace(expr)

	// Handle parentheses first
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[1 : len(expr)-1])
		return parseExpression(inner, ctx)
	}

	// Handle logical operators (&&, ||, !)
	if result, err := parseLogicalExpression(expr, ctx); err == nil {
		return result, nil
	}

	// Handle comparison operators (==, !=, <, <=, >, >=, =)
	if result, err := parseComparisonExpression(expr, ctx); err == nil {
		return result, nil
	}

	// Handle function calls
	if result, err := parseFunctionCall(expr, ctx); err == nil {
		return result, nil
	}

	// Handle simple variable references
	if result, err := parseVariableReference(expr, ctx); err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("unable to parse expression: %s", expr)
}

// parseLogicalExpression handles &&, ||, ! operators
func parseLogicalExpression(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	// Handle NOT operator
	if strings.HasPrefix(expr, "!") {
		operand := strings.TrimSpace(expr[1:])
		// Handle parentheses for NOT operator
		if strings.HasPrefix(operand, "(") && strings.HasSuffix(operand, ")") {
			operand = strings.TrimSpace(operand[1 : len(operand)-1])
		}
		result, err := parseExpression(operand, ctx)
		if err != nil {
			return nil, err
		}
		return &ExpressionResult{
			Value: !toBool(result),
			Type:  "bool",
		}, nil
	}

	// Handle AND operator - need to be careful about parentheses
	if strings.Contains(expr, " && ") {
		// Find the AND operator that's not inside parentheses
		parenCount := 0
		andPos := -1
		for i, char := range expr {
			if char == '(' {
				parenCount++
			} else if char == ')' {
				parenCount--
			} else if char == '&' && i+1 < len(expr) && expr[i+1] == '&' && parenCount == 0 {
				// Check if this is a complete && operator
				if i > 0 && expr[i-1] == ' ' && i+2 < len(expr) && expr[i+2] == ' ' {
					andPos = i
					break
				}
			}
		}

		if andPos != -1 {
			left := strings.TrimSpace(expr[:andPos])
			right := strings.TrimSpace(expr[andPos+2:])

			leftResult, err := parseExpression(left, ctx)
			if err != nil {
				return nil, err
			}

			rightResult, err := parseExpression(right, ctx)
			if err != nil {
				return nil, err
			}

			return &ExpressionResult{
				Value: toBool(leftResult) && toBool(rightResult),
				Type:  "bool",
			}, nil
		}
	}

	// Handle OR operator - need to be careful about parentheses
	if strings.Contains(expr, " || ") {
		// Find the OR operator that's not inside parentheses
		parenCount := 0
		orPos := -1
		for i, char := range expr {
			if char == '(' {
				parenCount++
			} else if char == ')' {
				parenCount--
			} else if char == '|' && i+1 < len(expr) && expr[i+1] == '|' && parenCount == 0 {
				// Check if this is a complete || operator
				if i > 0 && expr[i-1] == ' ' && i+2 < len(expr) && expr[i+2] == ' ' {
					orPos = i
					break
				}
			}
		}

		if orPos != -1 {
			left := strings.TrimSpace(expr[:orPos])
			right := strings.TrimSpace(expr[orPos+2:])

			leftResult, err := parseExpression(left, ctx)
			if err != nil {
				return nil, err
			}

			rightResult, err := parseExpression(right, ctx)
			if err != nil {
				return nil, err
			}

			return &ExpressionResult{
				Value: toBool(leftResult) || toBool(rightResult),
				Type:  "bool",
			}, nil
		}
	}

	return nil, fmt.Errorf("not a logical expression")
}

// parseComparisonExpression handles ==, !=, <, <=, >, >=, = operators
func parseComparisonExpression(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	operators := []string{"!=", " = ", "<=", ">=", "<", ">", "=="}

	for _, op := range operators {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			if len(parts) != 2 {
				continue
			}

			left, err := parseExpression(strings.TrimSpace(parts[0]), ctx)
			if err != nil {
				return nil, err
			}

			right, err := parseExpression(strings.TrimSpace(parts[1]), ctx)
			if err != nil {
				return nil, err
			}

			// Normalize operator
			normalizedOp := strings.TrimSpace(op)
			if normalizedOp == "=" {
				normalizedOp = "=="
			}

			result, err := compareValues(left, right, normalizedOp)
			if err != nil {
				return nil, err
			}

			return &ExpressionResult{
				Value: result,
				Type:  "bool",
			}, nil
		}
	}

	return nil, fmt.Errorf("not a comparison expression")
}

// parseFunctionCall handles function calls like contains(), startsWith(), etc.
func parseFunctionCall(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	// Match function pattern: functionName(arg1, arg2, ...)
	re := regexp.MustCompile(`^(\w+)\((.*)\)$`)
	matches := re.FindStringSubmatch(expr)
	if len(matches) != 3 {
		return nil, fmt.Errorf("not a function call: %s", expr)
	}

	name := matches[1]
	argsStr := matches[2]

	var args []*ExpressionResult
	if argsStr != "" {
		// Simple split for now, needs more robust parsing for nested functions/commas in strings
		argParts := strings.Split(argsStr, ",")
		for _, argPart := range argParts {
			arg, err := parseExpression(strings.TrimSpace(argPart), ctx)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	switch name {
	case "contains":
		if len(args) != 2 {
			return nil, fmt.Errorf("contains() expects 2 arguments, got %d", len(args))
		}
		haystack := toString(args[0])
		needle := toString(args[1])
		return &ExpressionResult{
			Value: strings.Contains(haystack, needle),
			Type:  "bool",
		}, nil
	case "startsWith":
		if len(args) != 2 {
			return nil, fmt.Errorf("startsWith() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		p := toString(args[1])
		return &ExpressionResult{
			Value: strings.HasPrefix(s, p),
			Type:  "bool",
		}, nil
	case "endsWith":
		if len(args) != 2 {
			return nil, fmt.Errorf("endsWith() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		p := toString(args[1])
		return &ExpressionResult{
			Value: strings.HasSuffix(s, p),
			Type:  "bool",
		}, nil
	case "matches":
		if len(args) != 2 {
			return nil, fmt.Errorf("matches() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		reStr := toString(args[1])
		re, err := regexp.Compile(reStr)
		if err != nil {
			return nil, fmt.Errorf("invalid regex in matches(): %v", err)
		}
		return &ExpressionResult{
			Value: re.MatchString(s),
			Type:  "bool",
		}, nil
	case "fileExists":
		if len(args) != 1 {
			return nil, fmt.Errorf("fileExists() expects 1 argument, got %d", len(args))
		}
		path := toString(args[0])
		_, err := os.Stat(path)
		return &ExpressionResult{
			Value: !os.IsNotExist(err),
			Type:  "bool",
		}, nil
	case "semverCompare":
		if len(args) != 2 {
			return nil, fmt.Errorf("semverCompare() expects 2 arguments, got %d", len(args))
		}
		version1 := toString(args[0])
		version2 := toString(args[1])

		v1, err := version.Parse(version1)
		if err != nil {
			return nil, fmt.Errorf("invalid version 1: %v", err)
		}
		v2, err := version.Parse(version2)
		if err != nil {
			return nil, fmt.Errorf("invalid version 2: %v", err)
		}

		comparison := version.Compare(v1, v2)
		return &ExpressionResult{
			Value: comparison,
			Type:  "number",
		}, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

// compareValues compares two values using the specified operator
func compareValues(left, right *ExpressionResult, op string) (bool, error) {
	// Handle numeric comparison first (if both are numbers or one is number and other can be converted)
	if left.Type == "number" && right.Type == "number" {
		leftNum := toNumber(left)
		rightNum := toNumber(right)

		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		}
	}

	// Handle mixed numeric/string comparison
	if (left.Type == "number" || right.Type == "number") &&
		(left.Type == "string" || right.Type == "string") {
		leftNum := toNumber(left)
		rightNum := toNumber(right)

		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		}
	}

	// Handle string comparison (lexicographic)
	if left.Type == "string" || right.Type == "string" {
		leftStr := toString(left)
		rightStr := toString(right)

		switch op {
		case "==":
			return leftStr == rightStr, nil
		case "!=":
			return leftStr != rightStr, nil
		case "<":
			return leftStr < rightStr, nil
		case "<=":
			return leftStr <= rightStr, nil
		case ">":
			return leftStr > rightStr, nil
		case ">=":
			return leftStr >= rightStr, nil
		}
	}

	// Handle boolean comparison
	if left.Type == "bool" && right.Type == "bool" {
		leftBool := toBool(left)
		rightBool := toBool(right)

		switch op {
		case "==":
			return leftBool == rightBool, nil
		case "!=":
			return leftBool != rightBool, nil
		}
	}

	return false, fmt.Errorf("cannot compare %s and %s", left.Type, right.Type)
}

// Helper functions for type conversion
func toBool(result *ExpressionResult) bool {
	if result == nil {
		return false
	}

	switch v := result.Value.(type) {
	case bool:
		return v
	case string:
		// For strings, return true if non-empty (unless it's "false")
		if v == "false" {
			return false
		}
		return v != ""
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	default:
		return false
	}
}

func toString(result *ExpressionResult) string {
	if result == nil {
		return ""
	}

	switch v := result.Value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toNumber(result *ExpressionResult) float64 {
	if result == nil {
		return 0
	}

	switch v := result.Value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
		return 0
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Platform detection functions
func getOS() string {
	platformVars := GetPlatformVariables()
	return platformVars.OS
}

func getArch() string {
	platformVars := GetPlatformVariables()
	return platformVars.Arch
}

func getGitBranch() (string, error) {
	// This should use git commands to get the current branch
	// For now, return a simple implementation
	return "main", nil // This should be replaced with actual git branch detection
}

// parseVariableReference handles simple variable references
func parseVariableReference(expr string, ctx *ExpressionContext) (*ExpressionResult, error) {
	// Handle quoted strings
	if (strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) ||
		(strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) {
		value := expr[1 : len(expr)-1]
		return &ExpressionResult{
			Value: value,
			Type:  "string",
		}, nil
	}

	// Handle numbers
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return &ExpressionResult{
			Value: num,
			Type:  "number",
		}, nil
	}

	// Handle boolean literals
	if expr == "true" {
		return &ExpressionResult{
			Value: true,
			Type:  "bool",
		}, nil
	}
	if expr == "false" {
		return &ExpressionResult{
			Value: false,
			Type:  "bool",
		}, nil
	}

	// Handle variable references
	value, err := resolveVariable(expr, ctx)
	if err != nil {
		return nil, err
	}

	return &ExpressionResult{
		Value: value,
		Type:  "string",
	}, nil
}

// resolveVariable resolves a variable reference
func resolveVariable(name string, ctx *ExpressionContext) (string, error) {
	// Handle special variables - check context first, then fall back to platform detection
	switch name {
	case "os":
		if value, exists := ctx.Variables["os"]; exists {
			return value, nil
		}
		return getOS(), nil
	case "arch":
		if value, exists := ctx.Variables["arch"]; exists {
			return value, nil
		}
		return getArch(), nil
	case "ci":
		if ctx.CI {
			return "true", nil
		}
		return "false", nil
	case "branch":
		return ctx.Branch, nil
	}

	// Handle env.VAR syntax
	if strings.HasPrefix(name, "env.") {
		envVar := name[4:]
		return os.Getenv(envVar), nil
	}

	// Handle inputs.NAME syntax
	if strings.HasPrefix(name, "inputs.") {
		inputName := name[7:]
		if value, exists := ctx.Inputs[inputName]; exists {
			return value, nil
		}
		return "", fmt.Errorf("undefined input: %s", inputName)
	}

	// Handle matrix.os syntax
	if strings.HasPrefix(name, "matrix.") {
		matrixKey := name[7:]
		if value, exists := ctx.Matrix[matrixKey]; exists {
			return value, nil
		}
		return "", fmt.Errorf("undefined matrix key: %s", matrixKey)
	}

	// Handle regular variables
	if value, exists := ctx.Variables[name]; exists {
		return value, nil
	}

	return "", fmt.Errorf("undefined variable: %s", name)
}

// callFunction calls a built-in function
func callFunction(name string, args []*ExpressionResult) (*ExpressionResult, error) {
	switch name {
	case "contains":
		if len(args) != 2 {
			return nil, fmt.Errorf("contains() expects 2 arguments, got %d", len(args))
		}
		haystack := toString(args[0])
		needle := toString(args[1])
		return &ExpressionResult{
			Value: strings.Contains(haystack, needle),
			Type:  "bool",
		}, nil
	case "startsWith":
		if len(args) != 2 {
			return nil, fmt.Errorf("startsWith() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		p := toString(args[1])
		return &ExpressionResult{
			Value: strings.HasPrefix(s, p),
			Type:  "bool",
		}, nil
	case "endsWith":
		if len(args) != 2 {
			return nil, fmt.Errorf("endsWith() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		p := toString(args[1])
		return &ExpressionResult{
			Value: strings.HasSuffix(s, p),
			Type:  "bool",
		}, nil
	case "matches":
		if len(args) != 2 {
			return nil, fmt.Errorf("matches() expects 2 arguments, got %d", len(args))
		}
		s := toString(args[0])
		reStr := toString(args[1])
		re, err := regexp.Compile(reStr)
		if err != nil {
			return nil, fmt.Errorf("invalid regex in matches(): %v", err)
		}
		return &ExpressionResult{
			Value: re.MatchString(s),
			Type:  "bool",
		}, nil
	case "fileExists":
		if len(args) != 1 {
			return nil, fmt.Errorf("fileExists() expects 1 argument, got %d", len(args))
		}
		path := toString(args[0])
		_, err := os.Stat(path)
		return &ExpressionResult{
			Value: !os.IsNotExist(err),
			Type:  "bool",
		}, nil
	case "semverCompare":
		if len(args) != 2 {
			return nil, fmt.Errorf("semverCompare() expects 2 arguments, got %d", len(args))
		}
		version1 := toString(args[0])
		version2 := toString(args[1])

		v1, err := version.Parse(version1)
		if err != nil {
			return nil, fmt.Errorf("invalid version 1: %v", err)
		}
		v2, err := version.Parse(version2)
		if err != nil {
			return nil, fmt.Errorf("invalid version 2: %v", err)
		}

		comparison := version.Compare(v1, v2)
		return &ExpressionResult{
			Value: comparison,
			Type:  "number",
		}, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

// parseFunctionArguments parses function arguments
func parseFunctionArguments(argsStr string, ctx *ExpressionContext) ([]*ExpressionResult, error) {
	if argsStr == "" {
		return []*ExpressionResult{}, nil
	}

	// Simple split for now, needs more robust parsing for nested functions/commas in strings
	argParts := strings.Split(argsStr, ",")
	args := make([]*ExpressionResult, len(argParts))

	for i, argPart := range argParts {
		arg, err := parseExpression(strings.TrimSpace(argPart), ctx)
		if err != nil {
			return nil, err
		}
		args[i] = arg
	}

	return args, nil
}
