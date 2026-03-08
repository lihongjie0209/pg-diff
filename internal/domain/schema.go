package domain

type Schema struct {
	Name       string
	Extensions map[string]*Extension
	Types      map[string]*Type
	Sequences  map[string]*Sequence
	Tables     map[string]*Table
	Views      map[string]*View
	Functions  map[string]*Function
	Triggers   map[string]*Trigger
	Policies   map[string]*RLSPolicy
}

type Extension struct {
	Name    string
	Version string
}

type Type struct {
	Name       string
	Definition string // e.g. "ENUM ('a', 'b')" or "AS (a int, b text)"
}

type Sequence struct {
	Name       string
	Definition string // Contains increment, min/max, start, cache, cycle, owner if needed
}

type Function struct {
	Name       string
	Arguments  string // Full argument list, e.g. 'a integer, b text'
	Definition string // Full CREATE FUNCTION statement or body
	Language   string
	ReturnType string
}

type Table struct {
	Name        string
	Columns     map[string]*Column
	Constraints map[string]*Constraint
	Indices     map[string]*Index
	Privileges  map[string]string // Key: Grantee, Value: Comma-separated privileges (e.g. "SELECT, INSERT")
	Comment     *string           // Table Comment
}

type Column struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
	IsPrimaryKey bool
	Comment      *string // Column Comment
}

type Constraint struct {
	Name       string
	Type       string // CHECK, UNIQUE, FOREIGN KEY, etc.
	Definition string
}

type Index struct {
	Name       string
	Definition string
	IsUnique   bool
}

type View struct {
	Name       string
	Definition string
}

type Trigger struct {
	Name       string
	TableName  string
	Definition string
}

type RLSPolicy struct {
	Name      string
	TableName string
	Action    string // SELECT, INSERT, UPDATE, DELETE
	Roles     []string
	Using     string
	WithCheck string
}

// DiffRepository defines the interface for reading schema from DB
type DiffRepository interface {
	GetSchema(schemaName string) (*Schema, error)
}
