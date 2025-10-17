module github.com/AlexBurnes/buildfab/examples/test-buildfab-api

go 1.24.0

require github.com/AlexBurnes/buildfab v0.25.2

require (
	github.com/AlexBurnes/version-go v1.5.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/term v0.35.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/AlexBurnes/buildfab => ../..
