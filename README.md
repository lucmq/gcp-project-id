# google-cloud-id
[![Go Reference](https://pkg.go.dev/badge/github.com/lucmq/google-cloud-id.svg)](https://pkg.go.dev/github.com/lucmq/google-cloud-id)
[![Go Report Card](https://goreportcard.com/badge/github.com/lucmq/google-cloud-id)](https://goreportcard.com/report/github.com/lucmq/google-cloud-id)
[![Go Coverage](https://github.com/lucmq/google-cloud-id/wiki/coverage.svg)](https://raw.githack.com/wiki/lucmq/google-cloud-id/coverage.html)

Access your Google Cloud project ID and related configuration.

Works in the cloud and local development environments, as it can retrieve the project
ID configured within the `GCLOUD` CLI.

## Installation
To use this package in your Go project, you can install it using `go get`:

```bash
go get github.com/lucmq/google-cloud-id
```

## Usage
Here's how you can use the project package to retrieve a Google Cloud project ID:

```go
package main

import (
	"fmt"

	"github.com/lucmq/google-cloud-id/project"
)

func main() {
	// Retrieve the default project ID with default options
	projectID := project.ID()

	fmt.Println("Default Project ID:", projectID)
}
```

With custom options:

```go
package main

import (
	"fmt"
	"time"
	
	"github.com/lucmq/google-cloud-id/project"
)

func main() {
	// Customize options (e.g., timeout, scopes, strict mode)
	options := project.Options{
		Timeout: 10 * time.Second,
		Scopes:  []string{"https://www.googleapis.com/auth/compute"},
		Strict:  true,
	}

	// Retrieve the project ID with custom options
	projectID := project.ID(options)

	fmt.Println("Default Project ID:", projectID)
}
```

# Contributing
Contributions to this package are welcome! If you find any issues or have suggestions
for improvements, please feel free to open an issue or submit a pull request.

# License
This project is licensed under the MIT License - see the LICENSE file for details.
