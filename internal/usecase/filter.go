package usecase

import (
	"regexp"
	"strings"

	"github.com/user/pg-diff/internal/domain"
)

// FilterSchema returns a new Schema built from the original containing ONLY the listed object types
// (e.g. "table", "view", "function") and their recursive dependencies.
func FilterSchema(schema *domain.Schema, targetTypes []string) *domain.Schema {
	if len(targetTypes) == 0 {
		return schema
	}

	typesMap := make(map[string]bool)
	for _, t := range targetTypes {
		typesMap[strings.ToLower(strings.TrimSpace(t))] = true
	}

	required := make(map[string]bool)
	queue := []string{}

	enqueueAndRequire := func(name string) {
		if !required[name] {
			required[name] = true
			queue = append(queue, name)
		}
	}

	// Initial population based on types
	if typesMap["table"] {
		for name := range schema.Tables {
			enqueueAndRequire(name)
		}
	}
	if typesMap["view"] {
		for name := range schema.Views {
			enqueueAndRequire(name)
		}
	}
	if typesMap["function"] || typesMap["routine"] {
		for name := range schema.Functions {
			enqueueAndRequire(name)
		}
	}
	if typesMap["type"] || typesMap["enum"] {
		for name := range schema.Types {
			enqueueAndRequire(name)
		}
	}
	if typesMap["sequence"] {
		for name := range schema.Sequences {
			enqueueAndRequire(name)
		}
	}
	if typesMap["extension"] {
		for name := range schema.Extensions {
			enqueueAndRequire(name)
		}
	}
	if typesMap["trigger"] {
		for name := range schema.Triggers {
			enqueueAndRequire(name)
		}
	}
	if typesMap["policy"] || typesMap["rls"] {
		for name := range schema.Policies {
			enqueueAndRequire(name)
		}
	}

	// Regexes for parsing dependencies
	fkRegex := regexp.MustCompile(`(?i)REFERENCES\s+([a-zA-Z0-9_]+)`)
	funcCallRegex := regexp.MustCompile(`(?i)\b([a-zA-Z0-9_]+)\(`)

	processed := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if processed[current] {
			continue
		}
		processed[current] = true

		// Check if it's a table
		if table, ok := schema.Tables[current]; ok {
			// Scan Columns for Types and Sequences
			for _, col := range table.Columns {
				// Check for Enums/Custom Types
				if _, isType := schema.Types[col.DataType]; isType {
					if !required[col.DataType] {
						required[col.DataType] = true
						queue = append(queue, col.DataType)
					}
				}
				// Check Default Value for sequence (e.g. nextval('seq_name'))
				if col.DefaultValue != nil {
					if strings.Contains(*col.DefaultValue, "nextval") {
						for seqName := range schema.Sequences {
							if strings.Contains(*col.DefaultValue, "'"+seqName+"'") || strings.Contains(*col.DefaultValue, "'"+seqName+"::regclass'") {
								if !required[seqName] {
									required[seqName] = true
									queue = append(queue, seqName)
								}
							}
						}
					}
				}
			}

			// Scan constraints for Foreign Keys
			for _, constr := range table.Constraints {
				matches := fkRegex.FindStringSubmatch(constr.Definition)
				if len(matches) > 1 {
					fkTable := matches[1]
					if _, hasFK := schema.Tables[fkTable]; hasFK {
						if !required[fkTable] {
							required[fkTable] = true
							queue = append(queue, fkTable)
						}
					}
				}
			}

			// Triggers on this table: Require the functions they call
			for _, trig := range schema.Triggers {
				if trig.TableName == current {
					if !required[trig.Name] {
						required[trig.Name] = true
						queue = append(queue, trig.Name)
					}
					// Find the function it executes
					matches := funcCallRegex.FindAllStringSubmatch(trig.Definition, -1)
					for _, m := range matches {
						if len(m) > 1 {
							funcName := m[1]
							if _, hasFn := schema.Functions[funcName]; hasFn {
								if !required[funcName] {
									required[funcName] = true
									queue = append(queue, funcName)
								}
							}
						}
					}
				}
			}
		}

		// Check if it's a Function
		if fn, ok := schema.Functions[current]; ok {
			// A generic check inside function definition for other Tables/Views/Functions/Types
			// We iterate through all known entities and do a string match
			// (Heuristic boundary approach - could be over-eager but safe)
			body := fn.Definition + " " + fn.Arguments + " " + fn.ReturnType
			extractHeuristicDependencies(schema, body, required, &queue)
		}

		// Check if it's a View
		if view, ok := schema.Views[current]; ok {
			extractHeuristicDependencies(schema, view.Definition, required, &queue)
		}

		// Check if it's a Policy
		if policy, ok := schema.Policies[current]; ok {
			if !required[policy.TableName] {
				required[policy.TableName] = true
				queue = append(queue, policy.TableName)
			}
			extractHeuristicDependencies(schema, policy.Using, required, &queue)
			extractHeuristicDependencies(schema, policy.WithCheck, required, &queue)
		}
	}

	// Now build the new sub-schema
	filtered := &domain.Schema{
		Name:       schema.Name,
		Extensions: make(map[string]*domain.Extension),
		Types:      make(map[string]*domain.Type),
		Sequences:  make(map[string]*domain.Sequence),
		Tables:     make(map[string]*domain.Table),
		Views:      make(map[string]*domain.View),
		Functions:  make(map[string]*domain.Function),
		Triggers:   make(map[string]*domain.Trigger),
		Policies:   make(map[string]*domain.RLSPolicy),
	}

	for k, v := range schema.Extensions {
		if required[k] {
			filtered.Extensions[k] = v
		}
	}
	for k, v := range schema.Types {
		if required[k] {
			filtered.Types[k] = v
		}
	}
	for k, v := range schema.Sequences {
		if required[k] {
			filtered.Sequences[k] = v
		}
	}
	for k, v := range schema.Tables {
		if required[k] {
			filtered.Tables[k] = v
		}
	}
	for k, v := range schema.Views {
		if required[k] {
			filtered.Views[k] = v
		}
	}
	for k, v := range schema.Functions {
		if required[k] {
			filtered.Functions[k] = v
		}
	}
	for k, v := range schema.Triggers {
		if required[k] {
			filtered.Triggers[k] = v
		}
	}
	for k, v := range schema.Policies {
		if required[k] {
			filtered.Policies[k] = v
		}
	}

	return filtered
}

// extractHeuristicDependencies checks the provided string block against all known schema objects
// using regex word boundaries to pick up potential reliant objects correctly.
func extractHeuristicDependencies(schema *domain.Schema, block string, required map[string]bool, queue *[]string) {
	if block == "" {
		return
	}

	checkDep := func(name string) {
		if required[name] {
			return
		}

		// Use regex word boundary to ensure we don't partial match (e.g. matching 'status' inside 'user_status')
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `\b`)
		if re.MatchString(block) {
			required[name] = true
			*queue = append(*queue, name)
		}
	}

	for name := range schema.Tables {
		checkDep(name)
	}
	for name := range schema.Views {
		checkDep(name)
	}
	for name := range schema.Functions {
		checkDep(name)
	}
	for name := range schema.Types {
		checkDep(name)
	}
	for name := range schema.Sequences {
		checkDep(name)
	}
}
