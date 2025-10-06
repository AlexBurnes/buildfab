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
	
	// First pass: collect all variable references to validate they exist
	var missingVars []string
	allMatches := pattern.FindAllStringSubmatch(text, -1)
	
	for _, match := range allMatches {
		if len(match) >= 2 {
			varName := strings.TrimSpace(match[1])
			if _, exists := variables[varName]; !exists {
				missingVars = append(missingVars, varName)
			}
		}
	}
	
	// If any variables are missing, return a comprehensive error
	if len(missingVars) > 0 {
		var availableVars []string
		for varName := range variables {
			availableVars = append(availableVars, varName)
		}
		
		errorMsg := fmt.Sprintf("undefined variables: %s", strings.Join(missingVars, ", "))
		if len(availableVars) > 0 {
			errorMsg += fmt.Sprintf("\navailable variables: %s", strings.Join(availableVars, ", "))
		}
		return "", fmt.Errorf(errorMsg)
	}
	
	// Second pass: perform actual interpolation
	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name from ${{ variable }}
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Return original if no match
		}
		
		varName := strings.TrimSpace(submatches[1])
		
		// Variable should exist at this point since we validated above
		if value, exists := variables[varName]; exists {
			return value
		}
		
		// This should not happen due to validation above, but just in case
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
