package main

import (
	"fmt"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
	// Create runner with default options
	opts := buildfab.DefaultSimpleRunOptions()
	
	fmt.Println("=== Available Variables ===")
	for key, value := range opts.Variables {
		fmt.Printf("%s: %s\n", key, value)
	}
	
	// Test interpolation
	action := buildfab.Action{
		Name: "test",
		Run: "echo 'Project: ${{ version.project }}'",
	}
	
	interpolated, err := buildfab.InterpolateAction(action, opts.Variables)
	if err != nil {
		fmt.Printf("Interpolation error: %v\n", err)
		return
	}
	
	fmt.Printf("\nOriginal: %s\n", action.Run)
	fmt.Printf("Interpolated: %s\n", interpolated.Run)
}
