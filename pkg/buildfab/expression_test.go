package buildfab

import (
	"os"
	"testing"
)

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		variables map[string]string
		expected  bool
		expectErr bool
	}{
		// Basic variable references
		{
			name:      "simple variable reference",
			expr:      "os",
			variables: map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "empty variable reference",
			expr:      "empty_var",
			variables: map[string]string{"empty_var": ""},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "undefined variable",
			expr:      "undefined",
			variables: map[string]string{"os": "linux"},
			expected:  false,
			expectErr: true,
		},
		
		// Equality comparisons
		{
			name:      "equality match with ==",
			expr:      "os == 'ubuntu'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "equality no match with ==",
			expr:      "os == 'windows'",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "equality match with =",
			expr:      "arch = 'amd64'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "equality with double quotes",
			expr:      "platform == \"windows\"",
			variables: map[string]string{"platform": "windows"},
			expected:  true,
			expectErr: false,
		},
		
		// Inequality comparisons
		{
			name:      "inequality match with !=",
			expr:      "os != 'windows'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "inequality no match with !=",
			expr:      "os != 'ubuntu'",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// String comparisons (lexicographic)
		{
			name:      "string less than",
			expr:      "'apple' < 'banana'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "string greater than",
			expr:      "'zebra' > 'apple'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "string less than or equal",
			expr:      "'apple' <= 'apple'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "string greater than or equal",
			expr:      "'zebra' >= 'apple'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		
		// Numeric comparisons
		{
			name:      "numeric less than",
			expr:      "1 < 2",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "numeric greater than",
			expr:      "3 > 2",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "numeric equality",
			expr:      "5 == 5",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		
		// Boolean literals
		{
			name:      "boolean true",
			expr:      "true",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "boolean false",
			expr:      "false",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// Logical operators
		{
			name:      "AND operator - both true",
			expr:      "true && true",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "AND operator - one false",
			expr:      "true && false",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "OR operator - one true",
			expr:      "true || false",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "OR operator - both false",
			expr:      "false || false",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "NOT operator - true",
			expr:      "!false",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "NOT operator - false",
			expr:      "!true",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// Complex expressions
		{
			name:      "complex AND expression",
			expr:      "os == 'ubuntu' && arch == 'amd64'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "complex OR expression",
			expr:      "os == 'ubuntu' || os == 'darwin'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "complex NOT expression",
			expr:      "!(os == 'windows')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		
		// ${{ }} wrapper
		{
			name:      "with ${{ }} wrapper",
			expr:      "${{ os == 'ubuntu' }}",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		
		// Error cases
		{
			name:      "invalid syntax",
			expr:      "invalid syntax here",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		{
			name:      "unbalanced parentheses",
			expr:      "((true)",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExpressionContext(tt.variables)
			result, err := EvaluateExpression(tt.expr, ctx)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		variables map[string]string
		expected  bool
		expectErr bool
	}{
		// contains() function
		{
			name:      "contains - found",
			expr:      "contains('hello world', 'world')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "contains - not found",
			expr:      "contains('hello world', 'xyz')",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "contains - with variables",
			expr:      "contains(env.PATH, '/usr/bin')",
			variables: map[string]string{},
			expected:  true, // PATH should contain /usr/bin on most systems
			expectErr: false,
		},
		
		// startsWith() function
		{
			name:      "startsWith - true",
			expr:      "startsWith('hello world', 'hello')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "startsWith - false",
			expr:      "startsWith('hello world', 'world')",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// endsWith() function
		{
			name:      "endsWith - true",
			expr:      "endsWith('hello world', 'world')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "endsWith - false",
			expr:      "endsWith('hello world', 'hello')",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// matches() function
		{
			name:      "matches - valid regex",
			expr:      "matches('hello123', '^[a-z]+[0-9]+$')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "matches - invalid match",
			expr:      "matches('hello', '^[0-9]+$')",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "matches - invalid regex",
			expr:      "matches('hello', '[invalid')",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		
		// fileExists() function
		{
			name:      "fileExists - existing file",
			expr:      "fileExists('/dev/null')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "fileExists - non-existing file",
			expr:      "fileExists('/nonexistent/file')",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// semverCompare() function
		{
			name:      "semverCompare - equal versions",
			expr:      "semverCompare('1.0.0', '1.0.0') == 0",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "semverCompare - first greater",
			expr:      "semverCompare('1.1.0', '1.0.0') > 0",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "semverCompare - first less",
			expr:      "semverCompare('1.0.0', '1.1.0') < 0",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "semverCompare - invalid version",
			expr:      "semverCompare('invalid', '1.0.0')",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		
		// Function error cases
		{
			name:      "contains - wrong number of args",
			expr:      "contains('hello')",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		{
			name:      "unknown function",
			expr:      "unknownFunc('arg')",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExpressionContext(tt.variables)
			result, err := EvaluateExpression(tt.expr, ctx)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSpecialVariables(t *testing.T) {
	// Set up environment for testing
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")
	
	tests := []struct {
		name      string
		expr      string
		variables map[string]string
		inputs    map[string]string
		matrix    map[string]string
		ci        bool
		branch    string
		expected  bool
		expectErr bool
	}{
		// Environment variables
		{
			name:      "env variable - exists",
			expr:      "env.TEST_VAR == 'test_value'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "env variable - not exists",
			expr:      "env.NONEXISTENT == 'value'",
			variables: map[string]string{},
			expected:  false,
			expectErr: false,
		},
		
		// Input variables
		{
			name:      "input variable - exists",
			expr:      "inputs.name == 'test'",
			variables: map[string]string{},
			inputs:    map[string]string{"name": "test"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "input variable - not exists",
			expr:      "inputs.nonexistent == 'value'",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		
		// Matrix variables
		{
			name:      "matrix variable - exists",
			expr:      "matrix.os == 'linux'",
			variables: map[string]string{},
			matrix:    map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "matrix variable - not exists",
			expr:      "matrix.nonexistent == 'value'",
			variables: map[string]string{},
			expected:  false,
			expectErr: true,
		},
		
		// CI variable
		{
			name:      "ci variable - true",
			expr:      "ci == true",
			variables: map[string]string{},
			ci:        true,
			expected:  true,
			expectErr: false,
		},
		{
			name:      "ci variable - false",
			expr:      "ci == false",
			variables: map[string]string{},
			ci:        false,
			expected:  true,
			expectErr: false,
		},
		
		// Branch variable
		{
			name:      "branch variable - matches",
			expr:      "branch == 'main'",
			variables: map[string]string{},
			branch:    "main",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "branch variable - not matches",
			expr:      "branch == 'develop'",
			variables: map[string]string{},
			branch:    "main",
			expected:  false,
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExpressionContext(tt.variables)
			ctx.Inputs = tt.inputs
			ctx.Matrix = tt.matrix
			ctx.CI = tt.ci
			ctx.Branch = tt.branch
			
			result, err := EvaluateExpression(tt.expr, ctx)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		variables map[string]string
		expected  bool
		expectErr bool
	}{
		{
			name: "complex nested expression",
			expr: "os == 'ubuntu' && arch == 'amd64'",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name: "function with variables",
			expr: "contains(env.PATH, '/usr/bin') && startsWith(os, 'ubu')",
			variables: map[string]string{},
			expected:  true,
			expectErr: false,
		},
		{
			name: "multiple comparisons",
			expr: "semverCompare(version, '1.0.0') >= 0 && semverCompare(version, '2.0.0') < 0",
			variables: map[string]string{
				"version": "1.5.0",
			},
			expected:  true,
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExpressionContext(tt.variables)
			result, err := EvaluateExpression(tt.expr, ctx)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
