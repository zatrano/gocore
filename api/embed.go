// Package api, OpenAPI 3.1 spesifikasyonunu binary'ye gömer, böylece /openapi.yaml
// ve Swagger UI çalışma zamanında sunulabilir.
package api

import _ "embed"

// OpenAPISpec, gömülü OpenAPI 3.1 YAML spesifikasyonudur.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
