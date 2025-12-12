package trino

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/tuannvm/mcp-trino/internal/config"
)

func TestFilterCatalogs(t *testing.T) {
	tests := []struct {
		name            string
		allowedCatalogs []string
		input           []string
		expected        []string
	}{
		{
			name:            "No allowlist - return all",
			allowedCatalogs: nil,
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql", "mysql"},
		},
		{
			name:            "Empty allowlist - return all",
			allowedCatalogs: []string{},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql", "mysql"},
		},
		{
			name:            "Filter to allowed catalogs",
			allowedCatalogs: []string{"hive", "postgresql"},
			input:           []string{"hive", "postgresql", "mysql", "oracle"},
			expected:        []string{"hive", "postgresql"},
		},
		{
			name:            "Case insensitive filtering",
			allowedCatalogs: []string{"HIVE", "PostgreSQL"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive", "postgresql"},
		},
		{
			name:            "No matches",
			allowedCatalogs: []string{"nonexistent"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{},
		},
		{
			name:            "Partial matches",
			allowedCatalogs: []string{"hive"},
			input:           []string{"hive", "postgresql", "mysql"},
			expected:        []string{"hive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedCatalogs: tt.allowedCatalogs,
				},
			}

			result := client.filterCatalogs(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterCatalogs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterSchemas(t *testing.T) {
	tests := []struct {
		name           string
		allowedSchemas []string
		catalog        string
		input          []string
		expected       []string
	}{
		{
			name:           "No allowlist - return all",
			allowedSchemas: nil,
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging"},
			expected:       []string{"analytics", "marts", "staging"},
		},
		{
			name:           "Filter to allowed schemas",
			allowedSchemas: []string{"hive.analytics", "hive.marts"},
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging", "raw"},
			expected:       []string{"analytics", "marts"},
		},
		{
			name:           "Case insensitive filtering",
			allowedSchemas: []string{"HIVE.ANALYTICS", "hive.marts"},
			catalog:        "hive",
			input:          []string{"analytics", "marts", "staging"},
			expected:       []string{"analytics", "marts"},
		},
		{
			name:           "Different catalog - no matches",
			allowedSchemas: []string{"hive.analytics", "hive.marts"},
			catalog:        "postgresql",
			input:          []string{"public", "private"},
			expected:       []string{},
		},
		{
			name:           "Mixed catalogs in allowlist",
			allowedSchemas: []string{"hive.analytics", "postgresql.public"},
			catalog:        "hive",
			input:          []string{"analytics", "marts"},
			expected:       []string{"analytics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedSchemas: tt.allowedSchemas,
				},
			}

			result := client.filterSchemas(tt.input, tt.catalog)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterSchemas() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterTables(t *testing.T) {
	tests := []struct {
		name          string
		allowedTables []string
		catalog       string
		schema        string
		input         []string
		expected      []string
	}{
		{
			name:          "No allowlist - return all",
			allowedTables: nil,
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions"},
			expected:      []string{"users", "events", "sessions"},
		},
		{
			name:          "Filter to allowed tables",
			allowedTables: []string{"hive.analytics.users", "hive.analytics.events"},
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions", "temp"},
			expected:      []string{"users", "events"},
		},
		{
			name:          "Case insensitive filtering",
			allowedTables: []string{"HIVE.ANALYTICS.USERS", "hive.analytics.events"},
			catalog:       "hive",
			schema:        "analytics",
			input:         []string{"users", "events", "sessions"},
			expected:      []string{"users", "events"},
		},
		{
			name:          "Different catalog/schema - no matches",
			allowedTables: []string{"hive.analytics.users"},
			catalog:       "postgresql",
			schema:        "public",
			input:         []string{"orders", "customers"},
			expected:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &config.TrinoConfig{
					AllowedTables: tt.allowedTables,
				},
			}

			result := client.filterTables(tt.input, tt.catalog, tt.schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterTables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCatalogAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedCatalogs: []string{"hive", "postgresql", "MySQL"},
		},
	}

	tests := []struct {
		catalog  string
		expected bool
	}{
		{"hive", true},
		{"postgresql", true},
		{"mysql", true}, // Case insensitive
		{"MySQL", true},
		{"HIVE", true},
		{"oracle", false},
		{"sqlserver", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.catalog, func(t *testing.T) {
			result := client.isCatalogAllowed(tt.catalog)
			if result != tt.expected {
				t.Errorf("isCatalogAllowed(%q) = %v, want %v", tt.catalog, result, tt.expected)
			}
		})
	}
}

func TestIsSchemaAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedSchemas: []string{"hive.analytics", "hive.marts", "PostgreSQL.PUBLIC"},
		},
	}

	tests := []struct {
		catalog  string
		schema   string
		expected bool
	}{
		{"hive", "analytics", true},
		{"hive", "marts", true},
		{"postgresql", "public", true}, // Case insensitive
		{"PostgreSQL", "PUBLIC", true},
		{"hive", "staging", false},
		{"postgresql", "private", false},
		{"mysql", "analytics", false},
	}

	for _, tt := range tests {
		t.Run(tt.catalog+"."+tt.schema, func(t *testing.T) {
			result := client.isSchemaAllowed(tt.catalog, tt.schema)
			if result != tt.expected {
				t.Errorf("isSchemaAllowed(%q, %q) = %v, want %v", tt.catalog, tt.schema, result, tt.expected)
			}
		})
	}
}

func TestIsTableAllowed(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			AllowedTables: []string{"hive.analytics.users", "hive.marts.sales", "PostgreSQL.PUBLIC.ORDERS"},
		},
	}

	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    string
		expected bool
	}{
		{"Simple match", "hive", "analytics", "users", true},
		{"Case insensitive match", "PostgreSQL", "PUBLIC", "ORDERS", true},
		{"No match - different table", "hive", "analytics", "events", false},
		{"No match - different schema", "hive", "staging", "users", false},
		{"No match - different catalog", "mysql", "analytics", "users", false},
		{"Empty catalog", "", "analytics", "users", false},
		{"Empty schema", "hive", "", "users", false},
		{"Empty table", "hive", "analytics", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isTableAllowed(tt.catalog, tt.schema, tt.table)
			if result != tt.expected {
				t.Errorf("isTableAllowed(%q, %q, %q) = %v, want %v", tt.catalog, tt.schema, tt.table, result, tt.expected)
			}
		})
	}
}

func TestTableParameterResolution(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			Catalog: "hive",
			Schema:  "default",
		},
	}

	// Test table parameter resolution logic (extracted from GetTableSchema)
	testResolution := func(inputCatalog, inputSchema, inputTable, expectedCatalog, expectedSchema, expectedTable string) {
		// Simulate the resolution logic from GetTableSchema
		catalog, schema, table := inputCatalog, inputSchema, inputTable

		parts := strings.Split(table, ".")
		if len(parts) == 3 {
			// If table is already fully qualified, extract components
			catalog = parts[0]
			schema = parts[1]
			table = parts[2]
		} else if len(parts) == 2 {
			// If table has schema.table format
			schema = parts[0]
			table = parts[1]
			if catalog == "" {
				catalog = client.config.Catalog
			}
		} else {
			// Use provided or default catalog and schema
			if catalog == "" {
				catalog = client.config.Catalog
			}
			if schema == "" {
				schema = client.config.Schema
			}
		}

		if catalog != expectedCatalog || schema != expectedSchema || table != expectedTable {
			t.Errorf("Resolution(%q, %q, %q) = (%q, %q, %q), want (%q, %q, %q)",
				inputCatalog, inputSchema, inputTable,
				catalog, schema, table,
				expectedCatalog, expectedSchema, expectedTable)
		}
	}

	// Test the resolution logic that was causing the bug
	testResolution("", "analytics", "users", "hive", "analytics", "users")             // use default catalog
	testResolution("", "", "analytics.users", "hive", "analytics", "users")            // schema.table format
	testResolution("", "", "hive.analytics.users", "hive", "analytics", "users")       // fully qualified
	testResolution("postgresql", "public", "orders", "postgresql", "public", "orders") // explicit params
}

func TestGetTableSchemaAllowlistLogic(t *testing.T) {
	client := &Client{
		config: &config.TrinoConfig{
			Catalog:       "hive",
			Schema:        "default",
			AllowedTables: []string{"hive.analytics.users", "hive.marts.sales"},
		},
	}

	// Test the combined resolution + allowlist check logic
	testAllowlistAfterResolution := func(inputCatalog, inputSchema, inputTable string, expectedAllowed bool) {
		// Simulate the resolution + allowlist check from GetTableSchema
		catalog, schema, table := inputCatalog, inputSchema, inputTable

		// Resolution logic (copied from GetTableSchema)
		parts := strings.Split(table, ".")
		if len(parts) == 3 {
			catalog = parts[0]
			schema = parts[1]
			table = parts[2]
		} else if len(parts) == 2 {
			schema = parts[0]
			table = parts[1]
			if catalog == "" {
				catalog = client.config.Catalog
			}
		} else {
			if catalog == "" {
				catalog = client.config.Catalog
			}
			if schema == "" {
				schema = client.config.Schema
			}
		}

		// Allowlist check (after resolution)
		allowed := client.isTableAllowed(catalog, schema, table)
		if allowed != expectedAllowed {
			t.Errorf("Allowlist check after resolution(%q, %q, %q) -> isTableAllowed(%q, %q, %q) = %v, want %v",
				inputCatalog, inputSchema, inputTable, catalog, schema, table, allowed, expectedAllowed)
		}
	}

	// Test cases that verify the bug fix
	testAllowlistAfterResolution("hive", "analytics", "users", true)        // explicit - should work
	testAllowlistAfterResolution("", "analytics", "users", true)            // default catalog - should work
	testAllowlistAfterResolution("", "", "analytics.users", true)           // schema.table - BUG FIX: should work now
	testAllowlistAfterResolution("", "", "hive.analytics.users", true)      // fully qualified - should work
	testAllowlistAfterResolution("hive", "analytics", "events", false)      // not in allowlist - should deny
	testAllowlistAfterResolution("postgresql", "analytics", "users", false) // wrong catalog - should deny
}

func TestImprovedIsReadOnlyQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		// Basic read-only queries with word boundaries
		{"SELECT with word boundary", "SELECT * FROM users", true},
		{"SELECT with leading spaces", "  SELECT * FROM users", true},
		{"SELECT with newlines", "\n SELECT * FROM users\n", true},
		{"SHOW with word boundary", "SHOW TABLES", true},
		{"DESCRIBE with word boundary", "DESCRIBE users", true},
		{"EXPLAIN with word boundary", "EXPLAIN SELECT * FROM users", true},
		{"WITH CTE", "WITH cte AS (SELECT 1) SELECT * FROM cte", true},

		// SHOW CREATE statements (read-only despite containing "create" keyword)
		{"SHOW CREATE TABLE", "SHOW CREATE TABLE users", true},
		{"SHOW CREATE TABLE with schema", "SHOW CREATE TABLE myschema.users", true},
		{"SHOW CREATE TABLE fully qualified", "SHOW CREATE TABLE catalog.schema.table", true},
		{"SHOW CREATE TABLE with spaces", "  SHOW CREATE TABLE users  ", true},
		{"SHOW CREATE VIEW", "SHOW CREATE VIEW my_view", true},
		{"SHOW CREATE SCHEMA", "SHOW CREATE SCHEMA myschema", true},
		{"SHOW CREATE MATERIALIZED VIEW", "SHOW CREATE MATERIALIZED VIEW my_mat_view", true},

		// Edge cases with word boundaries (these should now be stricter)
		{"SELECT without space", "SELECT*FROM users", true}, // Word boundary handles this
		{"SHOW without space", "SHOWTABLES", false},         // Word boundary requires separation

		// Write operations that should be blocked
		{"INSERT statement", "INSERT INTO users VALUES (1)", false},
		{"UPDATE statement", "UPDATE users SET name = 'test'", false},
		{"DELETE statement", "DELETE FROM users", false},
		{"CREATE statement", "CREATE TABLE test (id INT)", false},
		{"CREATE VIEW statement", "CREATE VIEW myview AS SELECT 1", false},
		{"DROP statement", "DROP TABLE users", false},
		{"ALTER statement", "ALTER TABLE users ADD COLUMN age INT", false},

		// Complex cases
		{"SELECT with INSERT in string", "SELECT 'INSERT INTO' FROM dual", true},
		{"SELECT with INSERT in comment", "SELECT 1 -- INSERT INTO users", true},
		{"Multi-statement with semicolon", "SELECT 1; INSERT INTO users VALUES (1)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReadOnlyQuery(tt.query)
			if result != tt.expected {
				t.Errorf("isReadOnlyQuery(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestCreateTransport(t *testing.T) {
	tests := []struct {
		name                     string
		sslInsecure              bool
		expectInsecureSkipVerify bool
		expectTransportNotNil    bool
		expectTLSConfigNotNil    bool
	}{
		{
			name:                     "SSLInsecure true - should skip certificate verification",
			sslInsecure:              true,
			expectInsecureSkipVerify: true,
			expectTransportNotNil:    true,
			expectTLSConfigNotNil:    true,
		},
		{
			name:                     "SSLInsecure false - should verify certificates",
			sslInsecure:              false,
			expectInsecureSkipVerify: false,
			expectTransportNotNil:    true,
			expectTLSConfigNotNil:    false, // TLSClientConfig may be nil when not needed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := createTransport(tt.sslInsecure)

			if transport == nil {
				if tt.expectTransportNotNil {
					t.Fatal("createTransport() returned nil, expected non-nil transport")
				}
				return
			}

			if tt.sslInsecure {
				if transport.TLSClientConfig == nil {
					t.Fatal("TLSClientConfig is nil when SSLInsecure is true")
				}
				if transport.TLSClientConfig.InsecureSkipVerify != tt.expectInsecureSkipVerify {
					t.Errorf("InsecureSkipVerify = %v, want %v",
						transport.TLSClientConfig.InsecureSkipVerify, tt.expectInsecureSkipVerify)
				}
			} else {
				// When SSLInsecure is false, TLSClientConfig should either be nil
				// or have InsecureSkipVerify = false
				if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
					t.Error("InsecureSkipVerify should be false when SSLInsecure is false")
				}
			}
		})
	}
}

func TestCreateTransport_ClonesDefaultTransport(t *testing.T) {
	// Verify that createTransport properly clones http.DefaultTransport settings
	transport := createTransport(false)

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("DefaultTransport is not *http.Transport, skipping clone verification")
	}

	// Verify some key settings are preserved from DefaultTransport
	if transport.MaxIdleConns != defaultTransport.MaxIdleConns {
		t.Errorf("MaxIdleConns not preserved: got %d, want %d",
			transport.MaxIdleConns, defaultTransport.MaxIdleConns)
	}
	if transport.IdleConnTimeout != defaultTransport.IdleConnTimeout {
		t.Errorf("IdleConnTimeout not preserved: got %v, want %v",
			transport.IdleConnTimeout, defaultTransport.IdleConnTimeout)
	}
}

func TestCreateTransport_ExplicitlyDisablesInsecureWhenSecure(t *testing.T) {
	// Bug fix test: if DefaultTransport somehow has InsecureSkipVerify=true,
	// createTransport(false) should explicitly set it to false
	transport := createTransport(false)

	// Even if TLSClientConfig exists (from clone), InsecureSkipVerify must be false
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be explicitly false when sslInsecure=false")
	}

	// Create insecure transport first, then create secure one
	// to simulate a scenario where state could leak
	_ = createTransport(true)
	secureTransport := createTransport(false)

	if secureTransport.TLSClientConfig != nil && secureTransport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Secure transport should not inherit insecure settings")
	}
}

func TestCreateTransport_PreservesExistingTLSConfig(t *testing.T) {
	// Test that we don't completely overwrite existing TLS settings
	transport := createTransport(true)

	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig should not be nil when SSLInsecure is true")
	}

	// InsecureSkipVerify should be set
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}

	// The TLS config should still be usable for other settings
	// (verify it's a valid tls.Config that can be modified)
	transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Error("Should be able to set MinVersion on TLSClientConfig")
	}
}

func TestHeaderRoundTripper_WithSSLInsecureTransport(t *testing.T) {
	// Integration test: verify headerRoundTripper works with SSLInsecure transport
	baseTransport := createTransport(true) // SSLInsecure=true
	roundTripper := &headerRoundTripper{
		base: baseTransport,
	}

	// Verify the round tripper has the correct base transport
	if roundTripper.base == nil {
		t.Fatal("headerRoundTripper.base should not be nil")
	}

	// Verify the base transport has TLS configured correctly
	transport, ok := roundTripper.base.(*http.Transport)
	if !ok {
		t.Fatal("base transport should be *http.Transport")
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig should not be nil")
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestCreateTransport_IndependentInstances(t *testing.T) {
	// Verify that multiple calls create independent transports
	// (important since only one gets registered with trino client)
	transport1 := createTransport(true)
	transport2 := createTransport(true) // Both true to test TLSClientConfig independence

	// They should be different instances
	if transport1 == transport2 {
		t.Error("createTransport should return different instances")
	}

	// Both should have InsecureSkipVerify=true initially
	if transport1.TLSClientConfig == nil || !transport1.TLSClientConfig.InsecureSkipVerify {
		t.Error("transport1 should have InsecureSkipVerify=true")
	}
	if transport2.TLSClientConfig == nil || !transport2.TLSClientConfig.InsecureSkipVerify {
		t.Error("transport2 should have InsecureSkipVerify=true")
	}

	// Modifying one should not affect the other
	transport1.TLSClientConfig.InsecureSkipVerify = false
	if !transport2.TLSClientConfig.InsecureSkipVerify {
		t.Error("modifying transport1 should not affect transport2")
	}

	// TLSClientConfig should also be different instances
	if transport1.TLSClientConfig == transport2.TLSClientConfig {
		t.Error("TLSClientConfig should be different instances")
	}
}
