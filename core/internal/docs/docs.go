package docs

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

//go:embed openapi.json
var openapiJSON []byte

//go:embed swagger-ui.html
var swaggerHTML string

var (
	openapiYAML string
	once        sync.Once
)

func OpenAPIJSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(openapiJSON)
	}
}

func OpenAPIYAMLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			openapiYAML = jsonToYAML(openapiJSON)
		})
		w.Header().Set("Content-Type", "text/yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(openapiYAML))
	}
}

func DocsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerHTML))
	}
}

func jsonToYAML(data []byte) string {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		slog.Error("failed to unmarshal openapi for yaml conversion", "error", err)
		return "# error converting openapi spec"
	}
	return jsonToYAMLRecursive(obj, 0)
}

func jsonToYAMLRecursive(v interface{}, indent int) string {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case map[string]interface{}:
		var lines []string
		for k, child := range val {
			switch childVal := child.(type) {
			case map[string]interface{}, []interface{}:
				lines = append(lines, prefix+k+":")
				lines = append(lines, jsonToYAMLRecursive(childVal, indent+1))
			default:
				lines = append(lines, prefix+k+": "+jsonToYAMLRecursive(childVal, indent+1))
			}
		}
		return strings.Join(lines, "\n")
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		var items []string
		for _, item := range val {
			switch item.(type) {
			case map[string]interface{}, []interface{}:
				items = append(items, prefix+"- "+strings.TrimSpace(jsonToYAMLRecursive(item, indent+1)))
			default:
				items = append(items, prefix+"- "+jsonToYAMLRecursive(item, indent+1))
			}
		}
		return strings.Join(items, "\n")
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return ""
	}
}
