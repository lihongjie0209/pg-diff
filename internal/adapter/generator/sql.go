package generator

import (
	"fmt"
	"strings"

	"github.com/user/pg-diff/internal/domain"
	"github.com/user/pg-diff/internal/usecase"
)

type sqlGenerator struct{}

func NewSQLGenerator() usecase.Generator {
	return &sqlGenerator{}
}

func (g *sqlGenerator) GenerateCreateExtension(e *domain.Extension) string {
	return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s VERSION '%s';", e.Name, e.Version)
}

func (g *sqlGenerator) GenerateDropExtension(name string) string {
	return fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", name)
}

func (g *sqlGenerator) GenerateGrant(objType, objName, privileges, grantee string) string {
	return fmt.Sprintf("GRANT %s ON %s %s TO %s;", privileges, objType, objName, grantee)
}

func (g *sqlGenerator) GenerateRevoke(objType, objName, privileges, grantee string) string {
	return fmt.Sprintf("REVOKE %s ON %s %s FROM %s;", privileges, objType, objName, grantee)
}

func (g *sqlGenerator) GenerateCreateType(t *domain.Type) string {
	return fmt.Sprintf("CREATE TYPE %s %s;", t.Name, t.Definition)
}

func (g *sqlGenerator) GenerateDropType(name string) string {
	return fmt.Sprintf("DROP TYPE IF EXISTS %s;", name)
}

func (g *sqlGenerator) GenerateCreateSequence(s *domain.Sequence) string {
	return fmt.Sprintf("CREATE SEQUENCE %s %s;", s.Name, s.Definition)
}

func (g *sqlGenerator) GenerateAlterSequence(oldSeq, newSeq *domain.Sequence) string {
	// For simplicity, we just completely recreate or issue a broad ALTER Sequence
	// based off the new definition entirely.
	return fmt.Sprintf("ALTER SEQUENCE %s %s;", newSeq.Name, newSeq.Definition)
}

func (g *sqlGenerator) GenerateDropSequence(name string) string {
	return fmt.Sprintf("DROP SEQUENCE IF EXISTS %s;", name)
}

func (g *sqlGenerator) GenerateCreateTable(table *domain.Table) string {
	var cols []string
	for _, col := range table.Columns {
		nullability := "NOT NULL"
		if col.IsNullable {
			nullability = "NULL"
		}
		def := ""
		if col.DefaultValue != nil {
			def = fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
		}
		cols = append(cols, fmt.Sprintf("  %s %s %s%s", col.Name, col.DataType, nullability, def))
	}
	return fmt.Sprintf("CREATE TABLE %s (\n%s\n);", table.Name, strings.Join(cols, ",\n"))
}

func (g *sqlGenerator) GenerateDropTable(name string) string {
	return fmt.Sprintf("DROP TABLE %s;", name)
}

func (g *sqlGenerator) GenerateTableComment(tableName string, comment *string) string {
	if comment == nil {
		return fmt.Sprintf("COMMENT ON TABLE %s IS NULL;", tableName)
	}
	escapedComment := strings.ReplaceAll(*comment, "'", "''")
	return fmt.Sprintf("COMMENT ON TABLE %s IS '%s';", tableName, escapedComment)
}

func (g *sqlGenerator) GenerateAddColumn(tableName string, col *domain.Column) string {
	nullability := "NOT NULL"
	if col.IsNullable {
		nullability = "NULL"
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s %s;", tableName, col.Name, col.DataType, nullability)
}

func (g *sqlGenerator) GenerateDropColumn(tableName string, colName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, colName)
}

func (g *sqlGenerator) GenerateAlterColumn(tableName string, oldCol, newCol *domain.Column) string {
	var changes []string
	if oldCol.DataType != newCol.DataType {
		changes = append(changes, fmt.Sprintf("ALTER COLUMN %s TYPE %s", newCol.Name, newCol.DataType))
	}
	if oldCol.IsNullable != newCol.IsNullable {
		if newCol.IsNullable {
			changes = append(changes, fmt.Sprintf("ALTER COLUMN %s DROP NOT NULL", newCol.Name))
		} else {
			changes = append(changes, fmt.Sprintf("ALTER COLUMN %s SET NOT NULL", newCol.Name))
		}
	}
	return fmt.Sprintf("ALTER TABLE %s %s;", tableName, strings.Join(changes, ", "))
}

func (g *sqlGenerator) GenerateColumnComment(tableName string, colName string, comment *string) string {
	fullname := fmt.Sprintf("%s.%s", tableName, colName)
	if comment == nil {
		return fmt.Sprintf("COMMENT ON COLUMN %s IS NULL;", fullname)
	}
	escapedComment := strings.ReplaceAll(*comment, "'", "''")
	return fmt.Sprintf("COMMENT ON COLUMN %s IS '%s';", fullname, escapedComment)
}

func (g *sqlGenerator) GenerateAddConstraint(tableName string, c *domain.Constraint) string {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s %s;", tableName, c.Name, c.Definition)
}

func (g *sqlGenerator) GenerateDropConstraint(tableName string, constraintName string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;", tableName, constraintName)
}

func (g *sqlGenerator) GenerateCreateIndex(i *domain.Index) string {
	return fmt.Sprintf("%s;", i.Definition)
}

func (g *sqlGenerator) GenerateDropIndex(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s;", indexName)
}

func (g *sqlGenerator) GenerateCreateView(view *domain.View) string {
	return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s", view.Name, view.Definition)
}

func (g *sqlGenerator) GenerateDropView(name string) string {
	return fmt.Sprintf("DROP VIEW IF EXISTS %s;", name)
}

func (g *sqlGenerator) GenerateCreateTrigger(trigger *domain.Trigger) string {
	return fmt.Sprintf("CREATE TRIGGER %s %s", trigger.Name, trigger.Definition)
}

func (g *sqlGenerator) GenerateDropTrigger(name, tableName string) string {
	return fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", name, tableName)
}

func (g *sqlGenerator) GenerateCreatePolicy(policy *domain.RLSPolicy) string {
	roles := "PUBLIC"
	if len(policy.Roles) > 0 {
		roles = strings.Join(policy.Roles, ", ")
	}
	using := ""
	if policy.Using != "" {
		using = fmt.Sprintf(" USING (%s)", policy.Using)
	}
	withCheck := ""
	if policy.WithCheck != "" {
		withCheck = fmt.Sprintf(" WITH CHECK (%s)", policy.WithCheck)
	}

	return fmt.Sprintf("CREATE POLICY %s ON %s FOR %s TO %s%s%s;",
		policy.Name, policy.TableName, policy.Action, roles, using, withCheck)
}

func (g *sqlGenerator) GenerateDropPolicy(name, tableName string) string {
	return fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", name, tableName)
}

func (g *sqlGenerator) GenerateCreateFunction(fn *domain.Function) string {
	return fmt.Sprintf("CREATE OR REPLACE FUNCTION %s(%s) RETURNS %s AS $$\n%s\n$$ LANGUAGE %s;",
		fn.Name, fn.Arguments, fn.ReturnType, fn.Definition, fn.Language)
}

func (g *sqlGenerator) GenerateDropFunction(name string, args string) string {
	return fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s);", name, args)
}
