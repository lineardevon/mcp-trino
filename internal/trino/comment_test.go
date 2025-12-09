package trino

import (
	"testing"
)

func TestIsReadOnlyQueryWithComments(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name: "Single-line comment before SELECT",
			query: `-- This is a comment
SELECT * FROM table`,
			expected: true,
		},
		{
			name: "Multiple single-line comments",
			query: `-- Comment 1
-- Comment 2
SELECT id, name FROM users`,
			expected: true,
		},
		{
			name: "Multi-line comment before SELECT",
			query: `/* This is a
multi-line comment */
SELECT * FROM table`,
			expected: true,
		},
		{
			name: "Inline comment in SELECT",
			query: `SELECT /* inline comment */ * FROM table`,
			expected: true,
		},
		{
			name: "Comment before SHOW",
			query: `-- Get catalogs
SHOW CATALOGS`,
			expected: true,
		},
		{
			name: "Comment before WITH CTE",
			query: `-- CTE query
WITH temp AS (SELECT 1)
SELECT * FROM temp`,
			expected: true,
		},
		{
			name: "Comment before write operation should still fail",
			query: `-- This is dangerous
INSERT INTO table VALUES (1)`,
			expected: false,
		},
		{
			name: "Mixed comments and spaces",
			query: `
			
-- Comment
  /* another comment */
  
SELECT 1`,
			expected: true,
		},
		{
			name: "Comment containing apostrophe (DON'T)",
			query: `-- Bot code analysis - what happens if we DON'T filter out bots?
WITH bot_stats AS (
  SELECT bot_code,
    CASE 
      WHEN CONTAINS(xp, 'frontier-omni-fd') THEN 'xp'
    END as exp_group
  FROM pulse.sa.search_extended
)
SELECT * FROM bot_stats`,
			expected: true,
		},
		{
			name: "Comment with apostrophe and string literals",
			query: `-- This won't work without proper handling
SELECT * FROM users WHERE name = 'John'`,
			expected: true,
		},
		{
			name: "Multi-line comment with apostrophe",
			query: `/* Here's a comment
   that spans lines and won't
   break the parser */
SELECT 1`,
			expected: true,
		},
		{
			name: "Multiple apostrophes in comment",
			query: `-- It's important that we don't break when there's multiple apostrophes
SELECT id FROM table`,
			expected: true,
		},
		{
			name: "Double quotes in comment",
			query: `-- Use "double quotes" in identifiers
SELECT * FROM "table"`,
			expected: true,
		},
		{
			name: "Backticks in comment (Trino uses double quotes, not backticks)",
			query: "-- Use `backticks` for identifiers\nSELECT * FROM \"table\"",
			expected: true,
		},
		{
			name: "Mixed quotes in comment",
			query: `-- It's "complicated" with 'all' the quotes
SELECT 'value' FROM "table"`,
			expected: true,
		},
		{
			name: "Comment with unmatched quote at end of line",
			query: `-- This ends with a quote'
SELECT * FROM table`,
			expected: true,
		},
		{
			name: "Comment apostrophe followed by string literal on next line",
			query: `-- What if we DON'T do this?
WITH cte AS (SELECT 'value' as col) SELECT * FROM cte`,
			expected: true,
		},
		{
			name: "Write keyword in comment should still allow read query",
			query: `-- We could INSERT here but we won't
SELECT * FROM table`,
			expected: true,
		},
		{
			name: "Write keyword in comment should NOT allow actual write query",
			query: `-- This is a read query
INSERT INTO table VALUES (1)`,
			expected: false,
		},
		// Tests for comment markers inside string literals (state machine fix)
		{
			name:     "Comment marker inside string literal should be read-only",
			query:    "SELECT * FROM table WHERE msg = '-- test' OR id = 1",
			expected: true,
		},
		{
			name:     "Multi-line comment inside string literal should be read-only",
			query:    "SELECT * FROM table WHERE msg = '/* comment */'",
			expected: true,
		},
		{
			name:     "Write keyword inside string literal should be read-only",
			query:    "SELECT * FROM table WHERE msg = 'INSERT INTO test'",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReadOnlyQuery(tt.query)
			if result != tt.expected {
				t.Errorf("isReadOnlyQuery() = %v, want %v for query:\n%s", result, tt.expected, tt.query)
			}
		})
	}
}

func TestSanitizeQueryForKeywordDetection(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "Remove single-line comment",
			query:    "-- comment\nSELECT * FROM table",
			expected: "SELECT * FROM table",
		},
		{
			name:     "Remove multi-line comment",
			query:    "/* comment */SELECT * FROM table",
			expected: "SELECT * FROM table",
		},
		{
			name:     "Remove multiple comments",
			query:    "-- first\n/* second */SELECT * FROM table",
			expected: "SELECT * FROM table",
		},
		{
			name:     "Apostrophe in single-line comment preserved correctly",
			query:    "-- DON'T panic\nSELECT * FROM table WHERE name = 'John'",
			expected: "SELECT * FROM table WHERE name = 'LITERAL'",
		},
		{
			name:     "Multiple apostrophes in single-line comment",
			query:    "-- It's important that we don't break\nSELECT 1",
			expected: "SELECT 1",
		},
		{
			name:     "Apostrophe in multi-line comment",
			query:    "/* Here's a comment that won't break */\nSELECT 1",
			expected: "SELECT 1",
		},
		{
			name:     "String literal spanning would-be comment area is correctly handled",
			query:    "SELECT 'value' FROM table",
			expected: "SELECT 'LITERAL' FROM table",
		},
		{
			name:     "Complex query with comment containing quotes before string literals",
			query:    "-- We won't filter bots\nWITH cte AS (SELECT 'xp' as exp) SELECT * FROM cte",
			expected: "WITH cte AS (SELECT 'LITERAL' as exp) SELECT * FROM cte",
		},
		// Tests for comment markers inside string literals (state machine fix)
		{
			name:     "Single-line comment marker inside string literal",
			query:    "SELECT * FROM table WHERE msg = '-- test' OR id = 1",
			expected: "SELECT * FROM table WHERE msg = 'LITERAL' OR id = 1",
		},
		{
			name:     "Multi-line comment marker inside string literal",
			query:    "SELECT * FROM table WHERE msg = '/* comment */' AND id = 1",
			expected: "SELECT * FROM table WHERE msg = 'LITERAL' AND id = 1",
		},
		{
			name:     "Multiple comment markers inside string literal",
			query:    "SELECT '-- /* nested */ --' FROM t",
			expected: "SELECT 'LITERAL' FROM t",
		},
		{
			name:     "Comment marker at start of string literal",
			query:    "SELECT '--starts with dash' FROM t",
			expected: "SELECT 'LITERAL' FROM t",
		},
		{
			name:     "Real comment followed by string with comment marker",
			query:    "-- real comment\nSELECT '-- fake comment' FROM t",
			expected: "SELECT 'LITERAL' FROM t",
		},
		// Tests for unclosed multi-line comments
		{
			name:     "Unclosed multi-line comment at end",
			query:    "/* unclosed comment",
			expected: "",
		},
		{
			name:     "Unclosed multi-line comment after valid SQL",
			query:    "SELECT 1 /* unclosed",
			expected: "SELECT 1",
		},
		{
			name:     "Unclosed multi-line comment with trailing characters",
			query:    "SELECT 1 /* unclosed XYZ",
			expected: "SELECT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeQueryForKeywordDetection(tt.query)
			if result != tt.expected {
				t.Errorf("sanitizeQueryForKeywordDetection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

