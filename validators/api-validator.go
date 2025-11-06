package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Status  string            `json:"status"`  // SUCCESS or FAILED
	Message string            `json:"message"` // Human-readable message
	Details map[string]string `json:"details"` // Additional details
}

// APIConfig represents the APIs to validate
type APIConfig struct {
	RequiredAPIs []string `json:"requiredAPIs"`
	Project      string   `json:"project"`
}

// getMockGCPAPIs returns simulated enabled APIs based on force-failure setting
func getMockGCPAPIs(forceFailure bool) map[string]bool {
	if forceFailure {
		// Simulate API validation failure
		return map[string]bool{
			"compute.googleapis.com":           true,
			"container.googleapis.com":         true,
			"iam.googleapis.com":               false, // Intentionally disabled for demo
			"servicenetworking.googleapis.com": true,
		}
	}
	// Simulate API validation success
	return map[string]bool{
		"compute.googleapis.com":           true,
		"container.googleapis.com":         true,
		"iam.googleapis.com":               true, // All APIs enabled
		"servicenetworking.googleapis.com": true,
	}
}

func main() {
	// Read configuration from environment or use defaults
	project := os.Getenv("GCP_PROJECT")
	if project == "" {
		project = "demo-project"
	}

	requiredAPIsEnv := os.Getenv("REQUIRED_APIS")
	var requiredAPIs []string
	if requiredAPIsEnv != "" {
		requiredAPIs = strings.Split(requiredAPIsEnv, ",")
	} else {
		// Default required APIs
		requiredAPIs = []string{
			"compute.googleapis.com",
			"container.googleapis.com",
			"iam.googleapis.com",
		}
	}

	forceFailureEnv := os.Getenv("FORCE_FAILURE")
	forceFailure := forceFailureEnv == "true"

	// Perform validation
	result := validateAPIs(project, requiredAPIs, forceFailure)

	// Write results to files for Tekton
	writeResult(result)

	// Exit with appropriate code
	if result.Status == "FAILED" {
		os.Exit(1)
	}
	os.Exit(0)
}

func validateAPIs(project string, requiredAPIs []string, forceFailure bool) ValidationResult {
	var missingAPIs []string
	var enabledAPIs []string

	mockAPIs := getMockGCPAPIs(forceFailure)

	fmt.Printf("🔍 Validating API enablement for project: %s\n", project)
	fmt.Printf("📋 Checking %d required APIs...\n", len(requiredAPIs))
	fmt.Printf("⚙️  Force Failure: %t\n\n", forceFailure)

	for _, api := range requiredAPIs {
		enabled, exists := mockAPIs[api]
		if !exists || !enabled {
			fmt.Printf("❌ API %s: NOT ENABLED\n", api)
			missingAPIs = append(missingAPIs, api)
		} else {
			fmt.Printf("✅ API %s: ENABLED\n", api)
			enabledAPIs = append(enabledAPIs, api)
		}
	}

	fmt.Println()

	details := map[string]string{
		"project":      project,
		"enabled_apis": strings.Join(enabledAPIs, ","),
		"total_checks": fmt.Sprintf("%d", len(requiredAPIs)),
	}

	if len(missingAPIs) > 0 {
		details["missing_apis"] = strings.Join(missingAPIs, ",")
		details["missing_count"] = fmt.Sprintf("%d", len(missingAPIs))

		return ValidationResult{
			Status:  "FAILED",
			Message: fmt.Sprintf("API validation failed: %d API(s) not enabled - %s", len(missingAPIs), strings.Join(missingAPIs, ", ")),
			Details: details,
		}
	}

	return ValidationResult{
		Status:  "SUCCESS",
		Message: fmt.Sprintf("All %d required APIs are enabled", len(requiredAPIs)),
		Details: details,
	}
}

func writeResult(result ValidationResult) {
	// Write status to Tekton result file
	statusFile := "/tekton/results/status"
	if err := os.WriteFile(statusFile, []byte(result.Status), 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write status file: %v\n", err)
	}

	// Write message to Tekton result file
	messageFile := "/tekton/results/message"
	if err := os.WriteFile(messageFile, []byte(result.Message), 0644); err != nil {
		fmt.Printf("⚠️  Warning: Could not write message file: %v\n", err)
	}

	// Write details as JSON to Tekton result file
	detailsFile := "/tekton/results/details"
	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not marshal details: %v\n", err)
	} else {
		if err := os.WriteFile(detailsFile, detailsJSON, 0644); err != nil {
			fmt.Printf("⚠️  Warning: Could not write details file: %v\n", err)
		}
	}

	// Print summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Validation Status: %s\n", result.Status)
	fmt.Printf("📝 Message: %s\n", result.Message)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
