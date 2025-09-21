package main

import (
	"context"
	"fmt"
	"github.com/AlexBurnes/buildfab/internal/version"
)

func main() {
	v := version.New()
	vars, err := v.GetVersionVariables(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println("Version variables:")
	for k, v := range vars {
		fmt.Printf("  %s: %s\n", k, v)
	}
}