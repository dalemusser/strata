// internal/domain/models/authmethods.go
package models

// AuthMethod represents an authentication method option for the UI.
type AuthMethod struct {
	Value string // The value stored in the database
	Label string // The display label in the UI
}

// AllAuthMethods contains all supported auth methods with their display labels.
// This is used for validation and as a reference for all possible values.
var AllAuthMethods = []AuthMethod{
	{Value: "trust", Label: "Trust"},
	{Value: "password", Label: "Password"},
	{Value: "email", Label: "Email Verification"},
	{Value: "google", Label: "Google"},
	// Add more auth methods as they are implemented:
	// {Value: "microsoft", Label: "Microsoft"},
	// {Value: "clever", Label: "Clever"},
	// {Value: "classlink", Label: "Classlink"},
}

// EnabledAuthMethods contains the auth methods currently available in the UI.
// Modify this list to control which options appear in Auth Method dropdowns.
var EnabledAuthMethods = []AuthMethod{
	{Value: "trust", Label: "Trust"},
	{Value: "password", Label: "Password"},
	{Value: "email", Label: "Email Verification"},
	{Value: "google", Label: "Google"},
}

// IsValidAuthMethod checks if a value is a valid auth method.
func IsValidAuthMethod(value string) bool {
	for _, m := range AllAuthMethods {
		if m.Value == value {
			return true
		}
	}
	return false
}

// IsEnabledAuthMethod checks if a value is an enabled auth method.
func IsEnabledAuthMethod(value string) bool {
	for _, m := range EnabledAuthMethods {
		if m.Value == value {
			return true
		}
	}
	return false
}

// EnabledAuthMethodValues returns just the values of enabled auth methods.
func EnabledAuthMethodValues() []string {
	values := make([]string, len(EnabledAuthMethods))
	for i, m := range EnabledAuthMethods {
		values[i] = m.Value
	}
	return values
}
