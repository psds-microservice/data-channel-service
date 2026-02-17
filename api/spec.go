package api

import _ "embed"

// OpenAPISpec — OpenAPI 2.0 (Swagger) for Data Channel API.
//
//go:embed openapi.json
var OpenAPISpec []byte
