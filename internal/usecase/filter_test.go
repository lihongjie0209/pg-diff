package usecase

import (
	"testing"

	"github.com/user/pg-diff/internal/domain"
)

func TestFilterSchema(t *testing.T) {
	// Build a mock schema
	schema := &domain.Schema{
		Name: "public",
		Types: map[string]*domain.Type{
			"user_status":  {Name: "user_status", Definition: "AS ENUM ('active', 'inactive')"},
			"other_status": {Name: "other_status", Definition: "AS ENUM ('foo')"},
		},
		Sequences: map[string]*domain.Sequence{
			"user_id_seq": {Name: "user_id_seq", Definition: "INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1"},
			"other_seq":   {Name: "other_seq", Definition: "INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START 1 CACHE 1"},
		},
		Tables: map[string]*domain.Table{
			"users": {
				Name: "users",
				Columns: map[string]*domain.Column{
					"id":     {Name: "id", DataType: "integer", DefaultValue: strPtr("nextval('user_id_seq'::regclass)")},
					"status": {Name: "status", DataType: "user_status"},
				},
				Constraints: map[string]*domain.Constraint{
					"fk_role": {Name: "fk_role", Type: "f", Definition: "FOREIGN KEY (role_id) REFERENCES roles(id)"},
				},
			},
			"roles": {
				Name: "roles",
				Columns: map[string]*domain.Column{
					"id": {Name: "id", DataType: "integer"},
				},
			},
			"isolated_table": {
				Name:    "isolated",
				Columns: map[string]*domain.Column{},
			},
		},
		Functions: map[string]*domain.Function{
			"get_user_status": {
				Name:       "get_user_status",
				Arguments:  "uid integer",
				ReturnType: "user_status",
				Definition: "BEGIN RETURN (SELECT status FROM users WHERE id = uid); END;",
			},
			"standalone_func": {
				Name:       "standalone_func",
				Definition: "BEGIN RETURN 1; END;",
			},
		},
		Views: map[string]*domain.View{
			"active_users": {
				Name:       "active_users",
				Definition: "SELECT * FROM users WHERE status = 'active'",
			},
		},
	}

	tests := []struct {
		name        string
		targetTypes []string
		wantTables  []string
		wantTypes   []string
		wantSeqs    []string
		wantFuncs   []string
		wantViews   []string
	}{
		{
			name:        "Empty targets returns all",
			targetTypes: []string{},
			wantTables:  []string{"users", "roles", "isolated_table"},
			wantTypes:   []string{"user_status", "other_status"},
			wantSeqs:    []string{"user_id_seq", "other_seq"},
			wantFuncs:   []string{"get_user_status", "standalone_func"},
			wantViews:   []string{"active_users"},
		},
		{
			name:        "Sync view only isolates view and underlying table dependencies",
			targetTypes: []string{"view"},
			wantViews:   []string{"active_users"},
			wantTables:  []string{"users", "roles"}, // active_users -> users -> roles
			wantTypes:   []string{"user_status"},    // users -> user_status
			wantSeqs:    []string{"user_id_seq"},    // users -> user_id_seq
			wantFuncs:   []string{},                 // none expected
		},
		{
			name:        "Sync table pulls explicit types, sequences, and FKs",
			targetTypes: []string{"table"},
			wantTables:  []string{"users", "roles", "isolated_table"},
			wantTypes:   []string{"user_status"},
			wantSeqs:    []string{"user_id_seq"},
			wantFuncs:   []string{},
			wantViews:   []string{},
		},
		{
			name:        "Sync functions pulls specific function and its referenced schema types",
			targetTypes: []string{"function"},
			wantFuncs:   []string{"get_user_status", "standalone_func"},
			wantTables:  []string{"users", "roles"}, // get_user_status -> users -> roles
			wantTypes:   []string{"user_status"},    // get_user_status -> user_status
			wantSeqs:    []string{"user_id_seq"},    // users -> user_id_seq
			wantViews:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterSchema(schema, tt.targetTypes)

			checkKeys("Tables", tt.wantTables, got.Tables, t)
			checkKeys("Types", tt.wantTypes, got.Types, t)
			checkKeys("Sequences", tt.wantSeqs, got.Sequences, t)
			checkKeys("Functions", tt.wantFuncs, got.Functions, t)
			checkKeys("Views", tt.wantViews, got.Views, t)
		})
	}
}

func checkKeys[T any](label string, expected []string, actualMap map[string]T, t *testing.T) {
	if len(actualMap) != len(expected) {
		t.Errorf("[%s] length mismatch: expected %d, got %d. Map: %v", label, len(expected), len(actualMap), getKeys(actualMap))
		return
	}
	for _, k := range expected {
		if _, ok := actualMap[k]; !ok {
			t.Errorf("[%s] missing expected key: %s. Map: %v", label, k, getKeys(actualMap))
		}
	}
}

func getKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func strPtr(s string) *string {
	return &s
}
