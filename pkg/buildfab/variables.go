package buildfab

import (
	"fmt"
	"regexp"
	"strings"
)

// InterpolateVariables replaces ${{ variable }} syntax with actual values
func InterpolateVariables(text string, variables map[string]string) (string, error) {
	if variables == nil {
		return text, nil
	}
	
	// Pattern to match ${{ variable }} syntax
	pattern := regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)
	
	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name from ${{ variable }}
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Return original if no match
		}
		
		varName := strings.TrimSpace(submatches[1])
		
		// Check if variable exists
		if value, exists := variables[varName]; exists {
			return value
		}
		
		// Return original match if variable not found
		return match
	})
	
	return result, nil
}

// InterpolateAction interpolates variables in an action's run command
func InterpolateAction(action Action, variables map[string]string) (Action, error) {
	if action.Run == "" {
		return action, nil
	}
	
	interpolated, err := InterpolateVariables(action.Run, variables)
	if err != nil {
		return action, fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	action.Run = interpolated
	return action, nil
}

// InterpolateStep interpolates variables in a step's configuration
func InterpolateStep(step Step, variables map[string]string) (Step, error) {
	// Interpolate the action name if it's a variable
	interpolatedAction, err := InterpolateVariables(step.Action, variables)
	if err != nil {
		return step, fmt.Errorf("failed to interpolate variables in step action: %w", err)
	}
	step.Action = interpolatedAction
	
	// Interpolate require dependencies
	for i, req := range step.Require {
		interpolated, err := InterpolateVariables(req, variables)
		if err != nil {
			return step, fmt.Errorf("failed to interpolate variables in step require %d: %w", i, err)
		}
		step.Require[i] = interpolated
	}
	
	// Interpolate if condition
	if step.If != "" {
		interpolated, err := InterpolateVariables(step.If, variables)
		if err != nil {
			return step, fmt.Errorf("failed to interpolate variables in step if condition: %w", err)
		}
		step.If = interpolated
	}
	
	return step, nil
}
