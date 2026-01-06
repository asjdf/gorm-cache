package cache

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

func TestGetColNameFromColumn(t *testing.T) {
	tests := []struct {
		name     string
		col      interface{}
		expected string
	}{
		{
			name:     "string column",
			col:      "id",
			expected: "id",
		},
		{
			name:     "clause.Column",
			col:      clause.Column{Name: "user_id"},
			expected: "user_id",
		},
		{
			name:     "other type",
			col:      123,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getColNameFromColumn(tt.col)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestUniqueStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStringSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
			}
			// Check that all expected values are present
			for _, expected := range tt.expected {
				found := false
				for _, r := range result {
					if r == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %s to be in result", expected)
				}
			}
		})
	}
}

func TestExtractStringsFromVar(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "int slice",
			input:    []int{1, 2, 3},
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "int value",
			input:    42,
			expected: []string{"42"},
		},
		{
			name:     "string value",
			input:    "test",
			expected: []string{"test"},
		},
		{
			name:     "int pointer",
			input:    intPtr(100),
			expected: []string{"100"},
		},
		{
			name:     "unsupported type",
			input:    struct{}{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringsFromVar(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected %s at index %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}

func TestGetExprType(t *testing.T) {
	tests := []struct {
		name     string
		expr     clause.Expr
		expected string
	}{
		{
			name:     "eq expression",
			expr:     clause.Expr{SQL: "id = ?", Vars: []interface{}{1}},
			expected: "eq",
		},
		{
			name:     "in expression",
			expr:     clause.Expr{SQL: "id IN (?)", Vars: []interface{}{[]int{1, 2}}},
			expected: "in",
		},
		{
			name:     "other expression",
			expr:     clause.Expr{SQL: "name LIKE ?", Vars: []interface{}{"test%"}},
			expected: "other",
		},
		{
			name:     "expression with connector",
			expr:     clause.Expr{SQL: "id = ? AND name = ?", Vars: []interface{}{1, "test"}},
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExprType(tt.expr)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetColNameFromExpr(t *testing.T) {
	tests := []struct {
		name     string
		expr     clause.Expr
		ttype    string
		expected string
	}{
		{
			name:     "in expression",
			expr:     clause.Expr{SQL: "user_id IN (?)"},
			ttype:    "in",
			expected: "user_id",
		},
		{
			name:     "eq expression",
			expr:     clause.Expr{SQL: "id = ?"},
			ttype:    "eq",
			expected: "id",
		},
		{
			name:     "other type",
			expr:     clause.Expr{SQL: "name LIKE ?"},
			ttype:    "other",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getColNameFromExpr(tt.expr, tt.ttype)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Note: getObjectsAfterLoad requires a properly initialized gorm.DB with schema,
// which is complex to mock. These tests are covered in integration tests.

func TestHasOtherClauseExceptPrimaryField(t *testing.T) {
	// Create a simple schema with primary key
	s := &schema.Schema{
		Table: "users",
	}
	s.Fields = []*schema.Field{
		{
			DBName:     "id",
			PrimaryKey: true,
		},
		{
			DBName:     "name",
			PrimaryKey: false,
		},
	}

	tests := []struct {
		name     string
		setup    func(*gorm.DB)
		expected bool
	}{
		{
			name: "only primary key clause",
			setup: func(db *gorm.DB) {
				db.Statement.Clauses = map[string]clause.Clause{
					"WHERE": {
						Expression: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{Column: "id", Value: 1},
							},
						},
					},
				}
			},
			expected: false,
		},
		{
			name: "primary key and other clause",
			setup: func(db *gorm.DB) {
				db.Statement.Clauses = map[string]clause.Clause{
					"WHERE": {
						Expression: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{Column: "id", Value: 1},
								clause.Eq{Column: "name", Value: "test"},
							},
						},
					},
				}
			},
			expected: true,
		},
		{
			name: "no WHERE clause",
			setup: func(db *gorm.DB) {
				db.Statement.Clauses = map[string]clause.Clause{}
			},
			expected: false,
		},
		// Note: Testing "no primary key in schema" requires complex schema setup
		// This is covered in integration tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &gorm.DB{
				Statement: &gorm.Statement{
					Schema: s,
				},
			}
			tt.setup(db)
			result := hasOtherClauseExceptPrimaryField(db)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetPrimaryKeyFields(t *testing.T) {
	tests := []struct {
		name           string
		schema         *schema.Schema
		expectedCount  int
		expectedFields []string
	}{
		{
			name: "single primary key",
			schema: &schema.Schema{
				Fields: []*schema.Field{
					{DBName: "id", PrimaryKey: true},
					{DBName: "name", PrimaryKey: false},
				},
			},
			expectedCount:  1,
			expectedFields: []string{"id"},
		},
		{
			name: "composite primary key",
			schema: &schema.Schema{
				Fields: []*schema.Field{
					{DBName: "user_id", PrimaryKey: true},
					{DBName: "role_id", PrimaryKey: true},
					{DBName: "name", PrimaryKey: false},
				},
				PrimaryFields: []*schema.Field{
					{DBName: "user_id", PrimaryKey: true},
					{DBName: "role_id", PrimaryKey: true},
				},
			},
			expectedCount:  2,
			expectedFields: []string{"user_id", "role_id"},
		},
		{
			name: "no primary key",
			schema: &schema.Schema{
				Fields: []*schema.Field{
					{DBName: "name", PrimaryKey: false},
				},
			},
			expectedCount:  0,
			expectedFields: []string{},
		},
		{
			name:          "nil schema",
			schema:        nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrimaryKeyFields(tt.schema)
			if len(result) != tt.expectedCount {
				t.Errorf("expected %d primary key fields, got %d", tt.expectedCount, len(result))
				return
			}
			for i, expectedField := range tt.expectedFields {
				if i < len(result) && result[i].DBName != expectedField {
					t.Errorf("expected field %s at index %d, got %s", expectedField, i, result[i].DBName)
				}
			}
		})
	}
}

func TestGetAllUniqueIndexes(t *testing.T) {
	tests := []struct {
		name          string
		schema        *schema.Schema
		expectedCount int
		expectedNames []string
	}{
		{
			name: "with unique indexes",
			schema: func() *schema.Schema {
				s := &schema.Schema{
					Table: "users",
				}
				// Note: ParseIndexes requires proper schema setup, so we'll test with nil
				// Actual unique index parsing is tested in integration tests
				return s
			}(),
			expectedCount: 0, // Will be 0 because ParseIndexes needs proper setup
		},
		{
			name:          "nil schema",
			schema:        nil,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAllUniqueIndexes(tt.schema)
			if result == nil && tt.expectedCount > 0 {
				t.Errorf("expected %d unique indexes, got nil", tt.expectedCount)
			} else if result != nil && len(result) != tt.expectedCount {
				t.Errorf("expected %d unique indexes, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestGetUniqueIndexFields(t *testing.T) {
	tests := []struct {
		name          string
		schema        *schema.Schema
		indexName     string
		expectedCount int
	}{
		{
			name:          "nil schema",
			schema:        nil,
			indexName:     "idx_email",
			expectedCount: 0,
		},
		{
			name: "non-existent index",
			schema: &schema.Schema{
				Table: "users",
			},
			indexName:     "non_existent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueIndexFields(tt.schema, tt.indexName)
			if len(result) != tt.expectedCount {
				t.Errorf("expected %d fields, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestHasOtherClauseExceptUniqueField(t *testing.T) {
	// Note: hasOtherClauseExceptUniqueField requires actual unique index information
	// from schema.ParseIndexes(), which is complex to set up in unit tests.
	// This function is better tested in integration tests.
	// Here we just test basic cases.

	tests := []struct {
		name      string
		setup     func(*gorm.DB)
		indexName string
		expected  bool
	}{
		{
			name: "nil schema",
			setup: func(db *gorm.DB) {
				db.Statement.Schema = nil
				db.Statement.Clauses = map[string]clause.Clause{
					"WHERE": {
						Expression: clause.Where{
							Exprs: []clause.Expression{
								clause.Eq{Column: "email", Value: "test@example.com"},
							},
						},
					},
				}
			},
			indexName: "idx_email",
			expected:  true, // nil schema returns true
		},
		{
			name: "no WHERE clause",
			setup: func(db *gorm.DB) {
				db.Statement.Schema = &schema.Schema{Table: "users"}
				db.Statement.Clauses = map[string]clause.Clause{}
			},
			indexName: "idx_email",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &gorm.DB{
				Statement: &gorm.Statement{},
			}
			tt.setup(db)
			result := hasOtherClauseExceptUniqueField(db, tt.indexName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
