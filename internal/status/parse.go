// ABOUTME: Argv parsers matching the oracle's parse_* helpers — --workflow-dir,
// ABOUTME: --set, --where, --fields, --archive/--resolve/--short-id, id material.
package status

import (
	"fmt"
	"strings"
)

// parseWorkflowDir extracts --workflow-dir and returns (dir, remaining, err).
// Matches parse_workflow_dir.
func parseWorkflowDir(args []string) (string, []string, error) {
	var remaining []string
	workflowDir := ""
	i := 0
	for i < len(args) {
		if args[i] == "--workflow-dir" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--workflow-dir requires an argument")
			}
			workflowDir = args[i+1]
			i += 2
			continue
		}
		remaining = append(remaining, args[i])
		i++
	}
	return workflowDir, remaining, nil
}

// parseRootArg extracts --root. Matches parse_root_arg.
func parseRootArg(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--root" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("--root requires an argument")
			}
			return args[i+1], nil
		}
	}
	return "", nil
}

// parseSingleArg parses a flag taking a single non---- argument (--resolve,
// --short-id). label names the argument in the error. Matches parse_resolve_arg
// / parse_short_id_arg.
func parseSingleArg(args []string, flag, label string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a %s argument", flag, label)
			}
			ref := args[i+1]
			if ref == "" || strings.HasPrefix(ref, "--") {
				return "", fmt.Errorf("%s requires a %s argument", flag, label)
			}
			return ref, nil
		}
	}
	return "", nil
}

// parseArchiveArg parses --archive <slug>. Matches parse_archive_arg.
func parseArchiveArg(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--archive" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("--archive requires a slug argument")
			}
			slug := args[i+1]
			if slug == "" || strings.HasPrefix(slug, "--") {
				return "", fmt.Errorf("--archive requires a slug argument")
			}
			return slug, nil
		}
	}
	return "", nil
}

// parseFieldsOptions parses --fields and --all-fields. Returns (explicitFields,
// allFieldsFlag, err). Matches parse_fields_options.
func parseFieldsOptions(args []string) ([]string, bool, error) {
	var explicitFields []string
	explicitSet := false
	allFieldsFlag := false
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--fields":
			if i+1 >= len(args) {
				return nil, false, fmt.Errorf("--fields requires a comma-separated list of field names")
			}
			raw := args[i+1]
			if strings.TrimSpace(raw) == "" {
				return nil, false, fmt.Errorf("--fields requires a comma-separated list of field names")
			}
			var parsed []string
			seen := map[string]bool{}
			for _, name := range strings.Split(raw, ",") {
				name = strings.TrimSpace(name)
				if name == "" || seen[name] {
					continue
				}
				seen[name] = true
				parsed = append(parsed, name)
			}
			if len(parsed) == 0 {
				return nil, false, fmt.Errorf("--fields requires a comma-separated list of field names")
			}
			explicitFields = parsed
			explicitSet = true
			i += 2
			continue
		case "--all-fields":
			allFieldsFlag = true
			i++
			continue
		}
		i++
	}
	if explicitSet && allFieldsFlag {
		return nil, false, fmt.Errorf("--fields and --all-fields are mutually exclusive")
	}
	if !explicitSet {
		return nil, allFieldsFlag, nil
	}
	return explicitFields, allFieldsFlag, nil
}

// parseIDMaterialOptions parses --id-seed/--id-actor. Returns (seed, actor,
// flagsSeen, err). Matches parse_id_material_options. flagsSeen lists which
// flags appeared, used for the "only with --next-id" guard.
func parseIDMaterialOptions(args []string) (string, string, []string, error) {
	values := map[string]string{"--id-seed": "", "--id-actor": ""}
	seen := map[string]bool{}
	var order []string
	i := 0
	for i < len(args) {
		arg := args[i]
		if _, ok := values[arg]; ok {
			if seen[arg] {
				return "", "", nil, fmt.Errorf("%s may only be supplied once", arg)
			}
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("%s requires an argument", arg)
			}
			value := args[i+1]
			if strings.HasPrefix(value, "--") {
				return "", "", nil, fmt.Errorf("%s requires an argument", arg)
			}
			values[arg] = value
			seen[arg] = true
			order = append(order, arg)
			i += 2
			continue
		}
		i++
	}
	return values["--id-seed"], values["--id-actor"], order, nil
}

// whereFilter is a parsed --where clause. value==nil means presence/absence.
type whereFilter struct {
	field string
	op    string // "=" or "!="
	value *string
}

const whereSyntaxHelp = "--where requires an operator: use 'field = value', 'field != value', 'field !=' (non-empty), or 'field =' (empty)"

// parseWhereFilters parses all --where clauses. Matches parse_where_filters.
func parseWhereFilters(args []string) ([]whereFilter, error) {
	var filters []whereFilter
	i := 0
	for i < len(args) {
		if args[i] == "--where" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--where requires an argument")
			}
			whereArg := args[i+1]
			if strings.TrimSpace(whereArg) == "" {
				return nil, fmt.Errorf("--where argument cannot be empty")
			}
			var op, fieldPart, valuePart string
			if strings.Contains(whereArg, "!=") {
				op = "!="
				fieldPart, valuePart, _ = strings.Cut(whereArg, "!=")
			} else if strings.Contains(whereArg, "=") {
				op = "="
				fieldPart, valuePart, _ = strings.Cut(whereArg, "=")
			} else {
				return nil, fmt.Errorf("%s", whereSyntaxHelp)
			}
			field := strings.TrimSpace(fieldPart)
			if field == "" {
				return nil, fmt.Errorf("%s", whereSyntaxHelp)
			}
			value := strings.TrimSpace(valuePart)
			var valPtr *string
			if value != "" {
				v := value
				valPtr = &v
			}
			filters = append(filters, whereFilter{field: field, op: op, value: valPtr})
			i += 2
			continue
		}
		i++
	}
	return filters, nil
}

// applyFilters keeps entities matching all --where clauses. Matches
// apply_filters.
func applyFilters(entities []*entity, filters []whereFilter) []*entity {
	if len(filters) == 0 {
		return entities
	}
	var out []*entity
	for _, e := range entities {
		match := true
		for _, f := range filters {
			fieldVal := e.fields[f.field]
			if f.op == "!=" {
				if f.value == nil {
					if fieldVal == "" {
						match = false
					}
				} else if fieldVal == *f.value {
					match = false
				}
			} else { // "="
				if f.value == nil {
					if fieldVal != "" {
						match = false
					}
				} else if fieldVal != *f.value {
					match = false
				}
			}
			if !match {
				break
			}
		}
		if match {
			out = append(out, e)
		}
	}
	return out
}

// setUpdate is a parsed --set target.
type setUpdate struct {
	slug    string
	updates []fieldUpdate
}

// parseSetArgs parses --set <slug> field=value... Matches parse_set_args. A
// token starting with -- terminates the field list (truncation).
func parseSetArgs(args []string) (*setUpdate, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--set" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--set requires a slug argument")
			}
			slug := args[i+1]
			var updates []fieldUpdate
			j := i + 2
			for j < len(args) && !strings.HasPrefix(args[j], "--") {
				arg := args[j]
				if strings.Contains(arg, "=") {
					field, value, _ := strings.Cut(arg, "=")
					updates = append(updates, fieldUpdate{field: field, value: value, hasValue: true})
				} else if timestampFields[arg] {
					updates = append(updates, fieldUpdate{field: arg, hasValue: false})
				} else {
					return nil, fmt.Errorf(
						"bare field '%s' requires a value (use %s=value); only completed, started support auto-fill",
						arg, arg)
				}
				j++
			}
			if len(updates) == 0 {
				return nil, fmt.Errorf("--set requires at least one field=value argument")
			}
			return &setUpdate{slug: slug, updates: updates}, nil
		}
	}
	return nil, nil
}

// parseNewArg parses --new [--folder] <slug>. Returns slug or "". The optional
// --folder flag may sit between --new and the slug; the slug is the next token
// and must not itself be --prefixed. A --new with no slug is an error.
func parseNewArg(args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--new" {
			j := i + 1
			if j < len(args) && args[j] == "--folder" {
				j++
			}
			if j >= len(args) {
				return "", fmt.Errorf("--new requires a slug argument")
			}
			slug := args[j]
			if slug == "" || strings.HasPrefix(slug, "--") {
				return "", fmt.Errorf("--new requires a slug argument")
			}
			return slug, nil
		}
	}
	return "", nil
}
