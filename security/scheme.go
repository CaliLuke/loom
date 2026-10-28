/*
Package security contains the types used by the code generators to
secure Loom endpoint. It supports the following security schemes:

  - Basic security using usernames and passwords.
  - API key security using keys.
  - JWT security using JWT tokens.
  - OAuth2 security using OAuth2 tokens.
*/
package security

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

type (
	// BasicScheme represents the BasicAuth security scheme.
	// It consists of a simple username and password.
	BasicScheme struct {
		// Name is the scheme name defined in the design.
		Name string
		// Scopes holds a list of scopes for the scheme.
		Scopes []string
		// RequiredScopes holds required application permission or role names.
		// RoleFor names need not occur in Scopes. Validate checks all these
		// names against the authenticated principal's application-owned grants.
		RequiredScopes []string
	}

	// APIKeyScheme represents the API key security scheme.
	// It consists of a key which is used in authentication.
	APIKeyScheme struct {
		// Name is the scheme name defined in the design.
		Name string
		// Scopes holds a list of scopes for the scheme.
		Scopes []string
		// RequiredScopes holds required application permission or role names.
		// RoleFor names need not occur in Scopes. Validate checks all these
		// names against the authenticated principal's application-owned grants.
		RequiredScopes []string
	}

	// JWTScheme represents an API key based scheme with support
	// for scopes.
	JWTScheme struct {
		// Name is the scheme name defined in the design.
		Name string
		// Scopes holds a list of scopes for the scheme.
		Scopes []string
		// RequiredScopes holds required application permission or role names.
		// RoleFor names need not occur in Scopes. Validate checks all these
		// names against the authenticated principal's application-owned grants.
		RequiredScopes []string
	}

	// OAuth2Scheme represents the oauth2 security scheme.
	OAuth2Scheme struct {
		// Name is the scheme name defined in the design.
		Name string
		// Scopes lists distinct available names across the effective flow inventories.
		// It describes availability, not grants held by an authenticated token.
		Scopes []string
		// RequiredScopes holds a list of scopes which are required
		// by the scheme. It is a subset of Scopes field.
		RequiredScopes []string
		// Flows determine the oauth2 flows.
		Flows []*OAuthFlow
	}

	// OAuthFlow represents the OAuth2 flow defined by the scheme.
	OAuthFlow struct {
		// Type is the type of grant.
		Type string
		// AuthorizationURL to be used for implicit or authorizationCode flows.
		AuthorizationURL string
		// TokenURL to be used for password, clientCredentials or authorizationCode flows.
		TokenURL string
		// RefreshURL to be used for obtaining refresh token.
		RefreshURL string
		// Scopes retains this flow's scope names and descriptions. Nil uses
		// the enclosing scheme's shared Scopes names; a nonnil empty map
		// declares an empty inventory. Inventories do not grant permissions.
		Scopes map[string]string
	}

	// AuthBasicFunc is the function type that implements the basic auth
	// scheme of using username and password.
	// Checks must be read-only and safe to repeat within one invocation, including
	// stream admission, the complete endpoint, and protected middleware boundaries.
	AuthBasicFunc func(ctx context.Context, user, pass string, s *BasicScheme) (context.Context, error)

	// AuthAPIKeyFunc is the function type that implements the API key
	// scheme of using an API key.
	// Checks must be read-only and safe to repeat within one invocation, including
	// stream admission, the complete endpoint, and protected middleware boundaries.
	AuthAPIKeyFunc func(ctx context.Context, key string, s *APIKeyScheme) (context.Context, error)

	// AuthOAuth2Func is the function type that implements the OAuth2
	// scheme of using an OAuth2 token.
	// Checks must be read-only and safe to repeat within one invocation, including
	// stream admission, the complete endpoint, and protected middleware boundaries.
	AuthOAuth2Func func(ctx context.Context, token string, s *OAuth2Scheme) (context.Context, error)

	// AuthJWTFunc is the function type that implements the JWT
	// scheme of using a JWT token.
	// Checks must be read-only and safe to repeat within one invocation, including
	// stream admission, the complete endpoint, and protected middleware boundaries.
	AuthJWTFunc func(ctx context.Context, token string, s *JWTScheme) (context.Context, error)
)

// Validate returns a non-nil error if scopes does not contain all of
// Basic scheme's required scopes.
func (s *BasicScheme) Validate(scopes []string) error {
	return validateScopes(s.RequiredScopes, scopes)
}

// Validate returns a non-nil error if scopes does not contain all of
// APIKey scheme's required scopes.
func (s *APIKeyScheme) Validate(scopes []string) error {
	return validateScopes(s.RequiredScopes, scopes)
}

// Validate returns a non-nil error if scopes does not contain all of
// OAuth2 scheme's required scopes.
func (s *OAuth2Scheme) Validate(scopes []string) error {
	return validateScopes(s.RequiredScopes, scopes)
}

// Validate returns a non-nil error if scopes does not contain all of
// JWT scheme's required scopes.
func (s *JWTScheme) Validate(scopes []string) error {
	return validateScopes(s.RequiredScopes, scopes)
}

func validateScopes(expected, actual []string) error {
	var missing []string
	for _, r := range expected {
		found := slices.Contains(actual, r)
		if !found {
			missing = append(missing, r)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing scopes: %s", strings.Join(missing, ", "))
}
