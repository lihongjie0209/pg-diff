package usecase

import (
	"testing"

	"github.com/user/pg-diff/internal/domain"
)

// MockGenerator 实现了升级后的 Generator 接口
type MockGenerator struct{}

func (m *MockGenerator) GenerateCreateTable(t *domain.Table) string { return "CREATE TABLE " + t.Name }
func (m *MockGenerator) GenerateDropTable(name string) string       { return "DROP TABLE " + name }
func (m *MockGenerator) GenerateAddColumn(tableName string, col *domain.Column) string {
	return "ALTER TABLE " + tableName + " ADD " + col.Name
}
func (m *MockGenerator) GenerateDropColumn(tableName string, colName string) string {
	return "ALTER TABLE " + tableName + " DROP " + colName
}
func (m *MockGenerator) GenerateAlterColumn(tableName string, oldCol, newCol *domain.Column) string {
	return "ALTER TABLE " + tableName + " ALTER " + newCol.Name
}
func (m *MockGenerator) GenerateAddConstraint(tableName string, c *domain.Constraint) string {
	return "ALTER TABLE " + tableName + " ADD CONSTRAINT " + c.Name
}
func (m *MockGenerator) GenerateDropConstraint(tableName string, constraintName string) string {
	return "ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraintName
}
func (m *MockGenerator) GenerateCreateIndex(i *domain.Index) string {
	return "CREATE INDEX " + i.Name
}
func (m *MockGenerator) GenerateDropIndex(indexName string) string {
	return "DROP INDEX " + indexName
}

func (m *MockGenerator) GenerateCreateExtension(e *domain.Extension) string {
	return "CREATE EXTENSION " + e.Name
}
func (m *MockGenerator) GenerateDropExtension(name string) string { return "DROP EXTENSION " + name }
func (m *MockGenerator) GenerateGrant(objType, objName, privileges, grantee string) string {
	return "GRANT " + privileges
}
func (m *MockGenerator) GenerateRevoke(objType, objName, privileges, grantee string) string {
	return "REVOKE " + privileges
}

func (m *MockGenerator) GenerateCreateType(t *domain.Type) string { return "CREATE TYPE " + t.Name }
func (m *MockGenerator) GenerateDropType(name string) string      { return "DROP TYPE " + name }
func (m *MockGenerator) GenerateCreateSequence(s *domain.Sequence) string {
	return "CREATE SEQUENCE " + s.Name
}
func (m *MockGenerator) GenerateAlterSequence(oldSeq, newSeq *domain.Sequence) string {
	return "ALTER SEQUENCE " + newSeq.Name
}
func (m *MockGenerator) GenerateDropSequence(name string) string { return "DROP SEQUENCE " + name }

func (m *MockGenerator) GenerateCreateView(v *domain.View) string { return "CREATE VIEW " + v.Name }
func (m *MockGenerator) GenerateDropView(name string) string      { return "DROP VIEW " + name }
func (m *MockGenerator) GenerateCreateTrigger(t *domain.Trigger) string {
	return "CREATE TRIGGER " + t.Name
}
func (m *MockGenerator) GenerateDropTrigger(name, tableName string) string {
	return "DROP TRIGGER " + name + " ON " + tableName
}
func (m *MockGenerator) GenerateCreatePolicy(p *domain.RLSPolicy) string {
	return "CREATE POLICY " + p.Name
}
func (m *MockGenerator) GenerateDropPolicy(name, tableName string) string {
	return "DROP POLICY " + name + " ON " + tableName
}
func (m *MockGenerator) GenerateCreateFunction(fn *domain.Function) string {
	return "CREATE FUNCTION " + fn.Name
}
func (m *MockGenerator) GenerateDropFunction(name string, args string) string {
	return "DROP FUNCTION " + name + "(" + args + ")"
}
func (m *MockGenerator) GenerateTableComment(name string, c *string) string {
	return "COMMENT ON TABLE " + name
}
func (m *MockGenerator) GenerateColumnComment(t, c string, com *string) string {
	return "COMMENT ON COLUMN " + c
}

func TestDiffService_FunctionDependency(t *testing.T) {
	gen := &MockGenerator{}
	svc := NewDiffService(gen)

	source := &domain.Schema{
		Functions: make(map[string]*domain.Function),
		Triggers:  make(map[string]*domain.Trigger),
	}

	target := &domain.Schema{
		Functions: map[string]*domain.Function{
			"update_ts": {Name: "update_ts", Definition: "BEGIN...END", Language: "plpgsql", ReturnType: "trigger"},
		},
		Triggers: map[string]*domain.Trigger{
			"trig_update": {Name: "trig_update", TableName: "users", Definition: "BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_ts()"},
		},
	}

	actions, err := svc.Compare(source, target)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	// Verify priority: Function should come before Trigger
	if len(actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(actions))
	}

	if actions[0].Object != "FUNCTION" || actions[1].Object != "TRIGGER" {
		t.Errorf("Incorrect order: expected FUNCTION then TRIGGER, got %s then %s", actions[0].Object, actions[1].Object)
	}
}

func TestDiffService_IncrementalCompare(t *testing.T) {
	gen := &MockGenerator{}
	svc := NewDiffService(gen)

	source := &domain.Schema{
		Tables: map[string]*domain.Table{
			"users": {
				Name: "users",
				Columns: map[string]*domain.Column{
					"id":   {Name: "id", DataType: "int"},
					"name": {Name: "name", DataType: "varchar"},
					"old":  {Name: "old", DataType: "varchar"},
				},
			},
		},
		Functions: make(map[string]*domain.Function),
		Views:     make(map[string]*domain.View),
		Triggers:  make(map[string]*domain.Trigger),
		Policies:  make(map[string]*domain.RLSPolicy),
	}

	target := &domain.Schema{
		Tables: map[string]*domain.Table{
			"users": {
				Name: "users",
				Columns: map[string]*domain.Column{
					"id":    {Name: "id", DataType: "bigint"},
					"name":  {Name: "name", DataType: "varchar"},
					"email": {Name: "email", DataType: "varchar"},
				},
			},
		},
		Functions: make(map[string]*domain.Function),
		Views:     make(map[string]*domain.View),
		Triggers:  make(map[string]*domain.Trigger),
		Policies:  make(map[string]*domain.RLSPolicy),
	}

	actions, err := svc.Compare(source, target)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	foundAdd := false
	foundAlter := false

	for _, a := range actions {
		if a.Type == "ALTER" && a.Name == "users.email" {
			foundAdd = true
		}
		if a.Type == "ALTER" && a.Name == "users.id" {
			foundAlter = true
		}
	}

	if !foundAdd {
		t.Errorf("Expected ADD email action not found")
	}
	if !foundAlter {
		t.Errorf("Expected ALTER id action not found")
	}
}
