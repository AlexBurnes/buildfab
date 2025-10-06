package main

import (
	"context"
	"fmt"
	"github.com/AlexBurnes/buildfab/internal/version"
)

func main() {
	detector := version.New()
	ctx := context.Background()
	
	variables, err := detector.GetVersionVariables(ctx)
	if err != nil {
		fmt.Printf("Error getting version variables: %v\n", err)
		return
	}
	
	fmt.Println("=== Available Version Variables ===")
	for key, value := range variables {
		fmt.Printf("%s: %s\n", key, value)
	}
}
