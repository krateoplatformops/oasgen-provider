internal/tools/oas2jsonschema/
├───generator.go             # Main orchestrator and entry point
├───types.go                 # Core domain types used across the package
├───errors.go                # Custom error types
├───interfaces.go            # Core interfaces (Parser, OASDocument)
│
├───adapter/                 # Adapters to external libraries
│   ├───crdgen.go            # Adapter for the crdgen package
│   └───libopenapi.go        # Adapter for the libopenapi parser
│
├───builder/                 # Logic for building the different schemas
│   ├───configuration.go
│   ├───spec.go
│   └───status.go
│
├───util/                    # Helper and utility functions
│   ├───extractor.go
│   ├───helpers.go
│   └───reflection.go
│
└───validation/              # Schema validation logic
    └───validator.go