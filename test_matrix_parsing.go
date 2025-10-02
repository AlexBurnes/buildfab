package main

import (
	"fmt"
	"log"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
	// Load the test configuration
	cfg, err := buildfab.LoadConfig("tests/test_matrix_debug.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Get the stage
	stage, exists := cfg.GetStage("debug-matrix")
	if !exists {
		log.Fatalf("Stage not found")
	}
	
	// Print the stage steps
	fmt.Printf("Stage: debug-matrix\n")
	fmt.Printf("Steps: %+v\n", stage.Steps)
	
	// Check each step for matrix configuration
	for i, step := range stage.Steps {
		fmt.Printf("Step %d: %+v\n", i, step)
		if step.Matrix != nil {
			fmt.Printf("  Matrix config: %+v\n", step.Matrix)
		} else {
			fmt.Printf("  No matrix config\n")
		}
	}
}
