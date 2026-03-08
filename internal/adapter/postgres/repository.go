package postgres

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/user/pg-diff/internal/domain"
)

type postgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connStr string) (domain.DiffRepository, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return &postgresRepository{db: db}, nil
}

func (r *postgresRepository) GetSchema(schemaName string) (*domain.Schema, error) {
	schema := &domain.Schema{
		Name:       schemaName,
		Extensions: make(map[string]*domain.Extension),
		Types:      make(map[string]*domain.Type),
		Sequences:  make(map[string]*domain.Sequence),
		Tables:     make(map[string]*domain.Table),
		Views:      make(map[string]*domain.View),
		Functions:  make(map[string]*domain.Function),
		Triggers:   make(map[string]*domain.Trigger),
		Policies:   make(map[string]*domain.RLSPolicy),
	}

	// -1. Extensions (usually globally mapped but handled within scope)
	extensions, err := r.fetchExtensions()
	if err != nil {
		return nil, err
	}
	schema.Extensions = extensions

	// 0. Types
	types, err := r.fetchTypes(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Types = types

	// 0.1 Sequences
	sequences, err := r.fetchSequences(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Sequences = sequences

	// 1. Tables
	tables, err := r.fetchTables(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Tables = tables

	// 2. Views
	views, err := r.fetchViews(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Views = views

	// 3. Functions
	functions, err := r.fetchFunctions(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Functions = functions

	// 4. Triggers
	triggers, err := r.fetchTriggers(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Triggers = triggers

	// 5. Policies
	policies, err := r.fetchPolicies(schemaName)
	if err != nil {
		return nil, err
	}
	schema.Policies = policies

	return schema, nil
}

func (r *postgresRepository) fetchFunctions(schemaName string) (map[string]*domain.Function, error) {
	// Query using pg_proc to get actual arguments properly formatted
	query := `
		SELECT 
			p.proname AS routine_name,
			pg_get_function_arguments(p.oid) AS arguments,
			pg_get_functiondef(p.oid) AS routine_definition,
			l.lanname AS external_language,
			pg_get_function_result(p.oid) AS data_type
		FROM 
			pg_proc p
		JOIN 
			pg_namespace n ON p.pronamespace = n.oid
		JOIN 
			pg_language l ON p.prolang = l.oid
		WHERE 
			n.nspname = $1 AND p.prokind = 'f'
			AND NOT EXISTS (
				SELECT 1 FROM pg_depend d 
				WHERE d.classid = 'pg_proc'::regclass AND d.objid = p.oid AND d.deptype = 'e'
			)`

	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	functions := make(map[string]*domain.Function)
	for rows.Next() {
		var name, args, definition, lang, retType string
		if err := rows.Scan(&name, &args, &definition, &lang, &retType); err != nil {
			return nil, err
		}
		functions[name] = &domain.Function{
			Name:       name,
			Arguments:  args,
			Definition: definition,
			Language:   lang,
			ReturnType: retType,
		}
	}
	return functions, nil
}

func (r *postgresRepository) fetchViews(schemaName string) (map[string]*domain.View, error) {
	query := `
		SELECT v.table_name, v.view_definition 
		FROM information_schema.views v
		JOIN pg_class c ON v.table_name = c.relname
		JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = v.table_schema
		WHERE v.table_schema = $1 
		AND NOT EXISTS (
			SELECT 1 FROM pg_depend d 
			WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
		)
	`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := make(map[string]*domain.View)
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			return nil, err
		}
		views[name] = &domain.View{Name: name, Definition: definition}
	}
	return views, nil
}

func (r *postgresRepository) fetchTriggers(schemaName string) (map[string]*domain.Trigger, error) {
	query := `SELECT trigger_name, event_object_table, action_statement 
	          FROM information_schema.triggers WHERE trigger_schema = $1`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggers := make(map[string]*domain.Trigger)
	for rows.Next() {
		var name, table, definition string
		if err := rows.Scan(&name, &table, &definition); err != nil {
			return nil, err
		}
		triggers[name] = &domain.Trigger{Name: name, TableName: table, Definition: definition}
	}
	return triggers, nil
}

func (r *postgresRepository) fetchPolicies(schemaName string) (map[string]*domain.RLSPolicy, error) {
	query := `SELECT policyname, tablename, cmd, roles, qual, with_check 
	          FROM pg_policies WHERE schemaname = $1`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make(map[string]*domain.RLSPolicy)
	for rows.Next() {
		var name, table, action, using, withCheck string
		var roles []string
		if err := rows.Scan(&name, &table, &action, pq.Array(&roles), &using, &withCheck); err != nil {
			return nil, err
		}
		policies[name] = &domain.RLSPolicy{
			Name:      name,
			TableName: table,
			Action:    action,
			Roles:     roles,
			Using:     using,
			WithCheck: withCheck,
		}
	}
	return policies, nil
}

func (r *postgresRepository) fetchTables(schemaName string) (map[string]*domain.Table, error) {
	query := `
		SELECT t.table_name, obj_description(c.oid, 'pg_class') as comment
		FROM information_schema.tables t
		JOIN pg_class c ON t.table_name = c.relname
		JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = t.table_schema
		WHERE t.table_schema = $1 AND t.table_type = 'BASE TABLE'
		AND NOT EXISTS (
			SELECT 1 FROM pg_depend d 
			WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
		)
	`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make(map[string]*domain.Table)
	for rows.Next() {
		var name string
		var comment sql.NullString
		if err := rows.Scan(&name, &comment); err != nil {
			return nil, err
		}

		var commentPtr *string
		if comment.Valid {
			commentPtr = &comment.String
		}

		table := &domain.Table{
			Name:    name,
			Columns: make(map[string]*domain.Column),
			Comment: commentPtr,
		}

		// 1.1 Fetch Columns for this table
		columns, err := r.fetchColumns(schemaName, name)
		if err != nil {
			return nil, err
		}
		table.Columns = columns

		// 1.2 Fetch Constraints for this table
		constraints, err := r.fetchConstraints(schemaName, name)
		if err != nil {
			return nil, err
		}
		table.Constraints = constraints

		// 1.3 Fetch Indices for this table
		indices, err := r.fetchIndices(schemaName, name)
		if err != nil {
			return nil, err
		}
		table.Indices = indices

		// 1.4 Fetch Privileges for this table
		privileges, err := r.fetchTablePrivileges(schemaName, name)
		if err != nil {
			return nil, err
		}
		table.Privileges = privileges

		tables[name] = table
	}
	return tables, nil
}

func (r *postgresRepository) fetchColumns(schemaName, tableName string) (map[string]*domain.Column, error) {
	// Query uses pg_class and pg_attribute for precise col_description
	query := `
		SELECT 
			a.attname AS column_name, 
			format_type(a.atttypid, a.atttypmod) AS data_type, 
			NOT a.attnotnull AS is_nullable, 
			pg_get_expr(ad.adbin, ad.adrelid) AS column_default,
			col_description(c.oid, a.attnum) AS comment
		FROM pg_attribute a
		JOIN pg_class c ON a.attrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		LEFT JOIN pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
		WHERE n.nspname = $1 AND c.relname = $2 AND a.attnum > 0 AND NOT a.attisdropped
	`
	rows, err := r.db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]*domain.Column)
	for rows.Next() {
		var name, dataType string
		var isNullable bool
		var columnDefault sql.NullString
		var comment sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &columnDefault, &comment); err != nil {
			return nil, err
		}

		var def *string
		if columnDefault.Valid {
			def = &columnDefault.String
		}

		var commentPtr *string
		if comment.Valid {
			commentPtr = &comment.String
		}

		columns[name] = &domain.Column{
			Name:         name,
			DataType:     dataType,
			IsNullable:   isNullable,
			DefaultValue: def,
			Comment:      commentPtr,
		}
	}
	return columns, nil
}

func (r *postgresRepository) fetchConstraints(schemaName, tableName string) (map[string]*domain.Constraint, error) {
	query := `
		SELECT 
			c.conname AS constraint_name,
			c.contype AS constraint_type,
			pg_get_constraintdef(c.oid) AS constraint_definition
		FROM 
			pg_constraint c
		JOIN 
			pg_namespace n ON n.oid = c.connamespace
		JOIN 
			pg_class cl ON cl.oid = c.conrelid
		WHERE 
			n.nspname = $1 AND cl.relname = $2`

	rows, err := r.db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	constraints := make(map[string]*domain.Constraint)
	for rows.Next() {
		var name, contype, definition string
		if err := rows.Scan(&name, &contype, &definition); err != nil {
			return nil, err
		}

		constraints[name] = &domain.Constraint{
			Name:       name,
			Type:       contype,
			Definition: definition,
		}
	}
	return constraints, nil
}

func (r *postgresRepository) fetchIndices(schemaName, tableName string) (map[string]*domain.Index, error) {
	query := `
		SELECT 
			i.relname AS index_name,
			pg_get_indexdef(i.oid) AS index_definition,
			ix.indisunique AS is_unique
		FROM 
			pg_class t
		JOIN 
			pg_index ix ON t.oid = ix.indrelid
		JOIN 
			pg_class i ON i.oid = ix.indexrelid
		JOIN 
			pg_namespace n ON n.oid = t.relnamespace
		WHERE 
			t.relkind = 'r' AND n.nspname = $1 AND t.relname = $2`

	rows, err := r.db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indices := make(map[string]*domain.Index)
	for rows.Next() {
		var name, definition string
		var isUnique bool
		if err := rows.Scan(&name, &definition, &isUnique); err != nil {
			return nil, err
		}

		indices[name] = &domain.Index{
			Name:       name,
			Definition: definition,
			IsUnique:   isUnique,
		}
	}
	return indices, nil
}

func (r *postgresRepository) fetchTypes(schemaName string) (map[string]*domain.Type, error) {
	// For simplicity, we just fetch ENUM types in this version.
	// Composite types ('c') require joining with pg_attribute.
	query := `
		SELECT
			t.typname AS type_name,
			string_agg(quote_literal(e.enumlabel), ', ' ORDER BY e.enumsortorder) AS enum_values
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE n.nspname = $1 AND t.typtype = 'e'
		AND NOT EXISTS (
			SELECT 1 FROM pg_depend d 
			WHERE d.classid = 'pg_type'::regclass AND d.objid = t.oid AND d.deptype = 'e'
		)
		GROUP BY t.typname;
	`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := make(map[string]*domain.Type)
	for rows.Next() {
		var name, enumValues string
		if err := rows.Scan(&name, &enumValues); err != nil {
			return nil, err
		}

		types[name] = &domain.Type{
			Name:       name,
			Definition: "AS ENUM (" + enumValues + ")",
		}
	}
	return types, nil
}

func (r *postgresRepository) fetchSequences(schemaName string) (map[string]*domain.Sequence, error) {
	query := `
		SELECT
			s.sequence_name,
			s.increment,
			s.minimum_value,
			s.maximum_value,
			s.start_value,
			s.cycle_option
		FROM information_schema.sequences s
		JOIN pg_class c ON s.sequence_name = c.relname
		JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = s.sequence_schema
		WHERE s.sequence_schema = $1
		AND NOT EXISTS (
			SELECT 1 FROM pg_depend d 
			WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
		);
	`
	rows, err := r.db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sequences := make(map[string]*domain.Sequence)
	for rows.Next() {
		var name, increment, minVal, maxVal, startVal, cycleOption string
		if err := rows.Scan(&name, &increment, &minVal, &maxVal, &startVal, &cycleOption); err != nil {
			return nil, err
		}

		cycleStr := "NO CYCLE"
		if cycleOption == "YES" {
			cycleStr = "CYCLE"
		}

		definition := "INCREMENT " + increment + " MINVALUE " + minVal + " MAXVALUE " + maxVal + " START " + startVal + " " + cycleStr

		sequences[name] = &domain.Sequence{
			Name:       name,
			Definition: definition,
		}
	}
	return sequences, nil
}

func (r *postgresRepository) fetchExtensions() (map[string]*domain.Extension, error) {
	query := `
		SELECT 
			extname, 
			extversion 
		FROM pg_extension;
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	extensions := make(map[string]*domain.Extension)
	for rows.Next() {
		var name, version string
		if err := rows.Scan(&name, &version); err != nil {
			return nil, err
		}
		extensions[name] = &domain.Extension{
			Name:    name,
			Version: version,
		}
	}
	return extensions, nil
}

func (r *postgresRepository) fetchTablePrivileges(schemaName, tableName string) (map[string]string, error) {
	// Standard SQL extraction from information_schema
	query := `
		SELECT grantee, string_agg(privilege_type, ', ') AS privileges
		FROM information_schema.role_table_grants 
		WHERE table_schema = $1 AND table_name = $2
		GROUP BY grantee;
	`
	rows, err := r.db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	privileges := make(map[string]string)
	for rows.Next() {
		var grantee, privs string
		if err := rows.Scan(&grantee, &privs); err != nil {
			return nil, err
		}
		privileges[grantee] = privs
	}
	return privileges, nil
}
