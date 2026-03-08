# Strata PCO

A Strata component for reading Planning Center Services data from typed Go tasks.

This component exposes read-only Services operations for:

- listing service types
- listing plans for a service type in a date range
- fetching plan details with team assignments
- clearing stored Planning Center authorization

## Authentication

The component uses Planning Center single-user authentication with two separate prompts:

- application ID
- secret

On the first API call, the component reads both values from component `Storage`. If either value is missing, it prompts through the host with `RequestSecret(...)`, stores the answers in `Storage`, and reuses them on later calls.

`ClearAuthorization` blanks both stored values so the next request prompts again.

Planning Center documents single-user auth as HTTP Basic Auth using `application_id:secret` against the Services API.

## Exported Operations

The shared `definitions` package exposes:

- `ListServiceTypes`
- `ListPlans`
- `GetPlanDetails`
- `ClearAuthorization`

## Using It In A Strata App

Add the dependency to your app:

```bash
go get github.com/jacksonzamorano/strata@v1.0.0
go get github.com/jacksonzamorano/strata-pco@latest
```

Import the component into your runtime:

```go
package main

import (
	"os"
	"path"

	"github.com/jacksonzamorano/strata"
)

func main() {
	cd, _ := os.Getwd()

	rt := strata.NewRuntime([]strata.Task{
		// your tasks here
	}, strata.Import(
		strata.ImportLocal(path.Join(path.Dir(cd), "strata-pco")),
	))

	panic(rt.Start())
}
```

Call it from a task:

```go
package main

import (
	"time"

	pco "github.com/jacksonzamorano/strata-pco/definitions"
	"github.com/jacksonzamorano/strata"
)

func upcomingPlans(input strata.RouteTaskNoInput, ctx *strata.TaskContext) *strata.RouteResult {
	plans, ok := pco.ListPlans.Execute(ctx.Container, pco.ListPlansInput{
		ServiceTypeID: "12345",
		From:          time.Now(),
		To:            time.Now().Add(14 * 24 * time.Hour),
	})
	if !ok {
		return strata.RouteResultError("failed to fetch plans")
	}
	return strata.RouteResultSuccess(plans)
}
```

## Notes

- `Storage` is used intentionally for both credential values.
- This component is callable-only and does not emit Strata triggers.
- The Services API response shape follows JSON:API.

## Sources

- https://github.com/planningcenter/developers
- https://support.planningcenteronline.com/hc/en-us/articles/4694905331867-The-Planning-Center-API
