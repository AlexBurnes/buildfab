package main

import (
	"fmt"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
	// Test the simple variable reference
	variables := map[string]string{"os": "linux"}
	ctx := buildfab.NewExpressionContext(variables)
	
	fmt.Printf("Context variables: %+v\n", ctx.Variables)
	
	result, err := buildfab.EvaluateExpression("os", ctx)
	fmt.Printf("os expression result: %v, error: %v\n", result, err)
	
	// Test what the os variable resolves to
	osValue, err := buildfab.ParseExpression("os", ctx)
	fmt.Printf("os variable value: %+v, error: %v\n", osValue, err)
}
