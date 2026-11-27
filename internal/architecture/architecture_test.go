// Package architecture holds the test that keeps the layers of TuneTrail apart
// (ISSUE-230). It reads the source of every production package and fails when one
// of them imports something its layer must not depend on. The rules and the way
// to register an exception are described in DEVELOPMENT.md.
package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/ebrgr/TuneTrail"

// Packages are named by their directory relative to the module root.
const (
	pkgSQLite    = "internal/db/sqlite"
	pkgDB        = "internal/db"
	pkgIngest    = "internal/ingest"
	pkgModels    = "internal/models"
	pkgMBZ       = "internal/mbz"
	pkgAdminUI   = "internal/adminui"
	pkgAdminWeb  = "internal/adminweb"
	pkgWebUI     = "internal/webui"
	pkgWebAssets = "internal/webassets"
	pkgEngine    = "engine"
	pkgHandlers  = "engine/handlers"
	pkgImageURL  = "internal/imageurl"
)

const (
	ruleWeb          = "web-without-sqlite"
	ruleSQLiteOwner  = "only-engine-imports-sqlite"
	ruleSQL          = "sql-only-in-sqlite"
	ruleCore         = "core-without-adapters"
	ruleDomain       = "domain-without-interface"
	ruleSQLiteImport = "sqlite-without-new-business-imports"
	ruleLeaf         = "leaf-stays-narrow"
)

// sqlStatement is the target of a violation caused by SQL text in a string.
const sqlStatement = "SQL statement"

var ruleText = map[string]string{
	ruleWeb:          "the web packages must not reach internal/db/sqlite, directly or through another package",
	ruleSQLiteOwner:  "only engine, which assembles the program, imports internal/db/sqlite",
	ruleSQL:          "SQL statements and SQL drivers stay inside internal/db/sqlite",
	ruleCore:         "internal/ingest, internal/models and internal/db must not reach adapters or the web interface",
	ruleDomain:       "domain packages must not import adminui, webui or engine",
	ruleSQLiteImport: "internal/db/sqlite must not gain imports of business packages",
	ruleLeaf:         "a leaf package imports only the module packages listed for it in leafImports",
}

// sqliteBase lists the module packages internal/db/sqlite may import for good:
// contracts, data types and technical helpers. Anything else is a business
// package and needs an entry in exceptions, which should not grow.
var sqliteBase = []string{
	"db/migrations_sqlite",
	"internal/cfg",
	pkgDB,
	pkgImageURL,
	pkgIngest,
	"internal/logger",
	pkgModels,
	"internal/retry",
	"internal/utils",
}

// leafImports limits packages that sqliteBase lets the storage adapter use, so
// that they cannot bring a business package in behind it. Direct imports only.
var leafImports = map[string][]string{
	pkgImageURL: {"imagecache", pkgDB, pkgModels},
}

type exception struct {
	rule   string
	pkg    string // package that breaks the rule
	target string // imported package or sqlStatement
	reason string
}

// exceptions records what already broke a rule when the test was created.
// The list may only shrink: remove an entry as soon as its import goes away
// (the test fails while a stale entry remains). A new entry needs a reason and
// the approval of the user in the task that introduces it.
var exceptions = []exception{
	{ruleSQL, "internal/testfixtures", sqlStatement,
		"test support only: inserts and counts fixture rows through the store"},
	{ruleSQL, "scripts/issue133_fault", "database/sql",
		"development tool that injects a migration failure, not part of the program"},
	{ruleSQL, "scripts/issue133_fault", "modernc.org/sqlite",
		"development tool that injects a migration failure, not part of the program"},
	{ruleSQL, "scripts/issue133_fault", sqlStatement,
		"development tool that injects a migration failure, not part of the program"},

	{ruleSQLiteImport, pkgSQLite, "internal/dedup",
		"stores deduplication receipts and normalizes names with dedup"},
	{ruleSQLiteImport, pkgSQLite, "internal/enrichment",
		"runs the enrichment queue with the types, registry and ListenBrainz provider of enrichment"},
	{ruleSQLiteImport, pkgSQLite, pkgMBZ,
		"ingestion resolution takes MusicBrainz release data and its caller"},
	{ruleSQLiteImport, pkgSQLite, "internal/relay",
		"persists relay destinations, grants and outbox with the types and policies of relay"},
}

type sourceFile struct {
	pkg     string
	path    string
	imports []string // module packages are relative to the module root
	sql     []int    // lines holding SQL statements
}

type violation struct {
	rule, pkg, target string
	where             []string // files, lines or import chains that prove it
}

// sqlText matches the statements as this repository writes them, with upper case
// keywords, so that interface wording such as "Delete from library" is ignored.
// Lower case SQL escapes this check; the driver import rule still applies to it.
var sqlText = regexp.MustCompile(`(?s)^\s*SELECT\s` +
	`|\bSELECT\b.*\bFROM\b` +
	`|\bINSERT\s+(OR\s+[A-Z]+\s+)?INTO\b` +
	`|\bREPLACE\s+INTO\b` +
	`|\bUPDATE\s+(OR\s+[A-Z]+\s+)?\w+\s+SET\b` +
	`|\bDELETE\s+FROM\b` +
	`|\b(CREATE|ALTER|DROP)\s+((UNIQUE|TEMP|TEMPORARY|VIRTUAL)\s+)?(TABLE|INDEX|TRIGGER|VIEW)\b` +
	`|\bPRAGMA\s+\w`)

var sqlDrivers = []string{
	"database/sql",
	"modernc.org/sqlite",
	"github.com/mattn/go-sqlite3",
	"github.com/jackc/pgx",
	"github.com/lib/pq",
}

// within reports whether path is root or lives below it.
func within(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

// loadModule parses every non-test Go file of the module under root, whatever its
// build tags. Nested modules and the directories the go tool ignores are skipped.
func loadModule(root string) ([]sourceFile, error) {
	var files []sourceFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		ignored := strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if ignored || name == "testdata" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if ignored || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		file, err := parseFile(root, path)
		if err != nil {
			return err
		}
		files = append(files, file)
		return nil
	})
	return files, err
}

func parseFile(root, path string) (sourceFile, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return sourceFile{}, err
	}
	relative = filepath.ToSlash(relative)
	result := sourceFile{path: relative, pkg: filepath.ToSlash(filepath.Dir(relative))}

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return sourceFile{}, err
	}
	for _, spec := range parsed.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return sourceFile{}, fmt.Errorf("%s: import %s: %w", relative, spec.Path.Value, err)
		}
		result.imports = append(result.imports, strings.TrimPrefix(imported, modulePath+"/"))
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ImportSpec:
			return false
		case *ast.BasicLit:
			if node.Kind == token.STRING {
				if text, err := strconv.Unquote(node.Value); err == nil && sqlText.MatchString(text) {
					result.sql = append(result.sql, fileSet.Position(node.Pos()).Line)
				}
			}
		}
		return true
	})
	return result, nil
}

// check applies every rule and returns one violation per rule, package and target.
func check(files []sourceFile) []violation {
	found := map[[3]string]*violation{}
	report := func(rule, pkg, target, where string) {
		key := [3]string{rule, pkg, target}
		if found[key] == nil {
			found[key] = &violation{rule: rule, pkg: pkg, target: target}
		}
		found[key].where = append(found[key].where, where)
	}

	// The keys of graph are the packages of the module.
	graph := map[string][]string{}
	for _, file := range files {
		graph[file.pkg] = append(graph[file.pkg], file.imports...)
	}
	for _, file := range files {
		for _, target := range file.imports {
			inSQLite := within(file.pkg, pkgSQLite)
			if within(target, pkgSQLite) && !inSQLite && file.pkg != pkgEngine {
				report(ruleSQLiteOwner, file.pkg, target, file.path)
			}
			if !inSQLite && slices.ContainsFunc(sqlDrivers, func(driver string) bool { return within(target, driver) }) {
				report(ruleSQL, file.pkg, target, file.path)
			}
			if !isInterface(file.pkg) && isForbiddenToDomain(target) {
				report(ruleDomain, file.pkg, target, file.path)
			}
			_, local := graph[target]
			if inSQLite && local && !within(target, pkgSQLite) && !slices.Contains(sqliteBase, target) {
				report(ruleSQLiteImport, file.pkg, target, file.path)
			}
			if allowed, leaf := leafImports[file.pkg]; leaf && local && !slices.Contains(allowed, target) {
				report(ruleLeaf, file.pkg, target, file.path)
			}
		}
		if !within(file.pkg, pkgSQLite) {
			for _, line := range file.sql {
				report(ruleSQL, file.pkg, sqlStatement, fmt.Sprintf("%s:%d", file.path, line))
			}
		}
	}

	// These two rules also follow indirect imports: an excepted package must not
	// become a way around them.
	for _, pkg := range []string{pkgAdminUI, pkgWebUI, pkgAdminWeb, pkgHandlers} {
		for target, chain := range reachable(graph, pkg) {
			if within(target, pkgSQLite) {
				report(ruleWeb, pkg, target, chain)
			}
		}
	}
	for _, pkg := range []string{pkgIngest, pkgModels, pkgDB} {
		for target, chain := range reachable(graph, pkg) {
			if within(target, pkgSQLite) || within(target, pkgMBZ) || within(target, pkgEngine) ||
				target == pkgAdminUI || target == pkgWebUI || target == pkgAdminWeb || target == pkgWebAssets {
				report(ruleCore, pkg, target, chain)
			}
		}
	}

	result := make([]violation, 0, len(found))
	for _, v := range found {
		slices.Sort(v.where)
		result = append(result, *v)
	}
	slices.SortFunc(result, func(a, b violation) int {
		return strings.Compare(a.rule+" "+a.pkg+" "+a.target, b.rule+" "+b.pkg+" "+b.target)
	})
	return result
}

// isInterface reports whether pkg belongs to the layer that serves users and
// assembles the program. Every other package, new ones included, is domain.
// webui importing adminui is therefore allowed; ISSUE-232 studies it.
func isInterface(pkg string) bool {
	return within(pkg, pkgEngine) || pkg == pkgAdminUI || pkg == pkgWebUI || within(pkg, "cmd")
}

func isForbiddenToDomain(target string) bool {
	return within(target, pkgEngine) || target == pkgAdminUI || target == pkgWebUI
}

// reachable returns every package imported by start, directly or not, with the
// shortest import chain that leads to it.
func reachable(graph map[string][]string, start string) map[string]string {
	chains := map[string]string{}
	pending := []string{start}
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		prefix := chains[current]
		if current == start {
			prefix = start
		}
		for _, next := range graph[current] {
			if _, seen := chains[next]; seen || next == start {
				continue
			}
			chains[next] = prefix + " -> " + next
			pending = append(pending, next)
		}
	}
	return chains
}

// compare splits the result into violations nobody excepted and exceptions that
// no longer match any violation.
func compare(violations []violation, allowed []exception) (unexpected []violation, stale []exception) {
	used := make([]bool, len(allowed))
	for _, v := range violations {
		index := slices.IndexFunc(allowed, func(e exception) bool {
			return e.rule == v.rule && e.pkg == v.pkg && e.target == v.target
		})
		if index < 0 {
			unexpected = append(unexpected, v)
			continue
		}
		used[index] = true
	}
	for index, e := range allowed {
		if !used[index] {
			stale = append(stale, e)
		}
	}
	return unexpected, stale
}

func TestArchitecture(t *testing.T) {
	root := filepath.Join("..", "..")
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(module), "module "+modulePath+"\n") {
		t.Fatalf("go.mod does not declare %s; update modulePath", modulePath)
	}
	files, err := loadModule(root)
	if err != nil {
		t.Fatal(err)
	}

	// A renamed or moved package would otherwise make its rules pass in silence.
	packages := map[string]bool{}
	for _, file := range files {
		packages[file.pkg] = true
	}
	named := append([]string{pkgSQLite, pkgDB, pkgIngest, pkgModels, pkgMBZ, pkgAdminUI,
		pkgAdminWeb, pkgWebUI, pkgWebAssets, pkgEngine, pkgHandlers}, sqliteBase...)
	for leaf, allowed := range leafImports {
		named = append(append(named, leaf), allowed...)
	}
	for _, pkg := range named {
		if !packages[pkg] {
			t.Errorf("package %s is named by a rule but has no source; update the rules", pkg)
		}
	}

	unexpected, stale := compare(check(files), exceptions)
	for _, v := range unexpected {
		t.Errorf("[%s] %s -> %s: %s\n\t%s\n\tRemove the dependency; see DEVELOPMENT.md before adding an exception.",
			v.rule, v.pkg, v.target, ruleText[v.rule], strings.Join(v.where, "\n\t"))
	}
	for _, e := range stale {
		t.Errorf("[%s] %s -> %s no longer happens: delete this entry from exceptions", e.rule, e.pkg, e.target)
	}
	for _, e := range exceptions {
		if ruleText[e.rule] == "" || strings.TrimSpace(e.reason) == "" {
			t.Errorf("exception %s -> %s needs a known rule and a reason", e.pkg, e.target)
		}
	}
}

// The cases below keep proof that each rule rejects the import it exists for.
func TestRulesRejectForbiddenImports(t *testing.T) {
	imports := func(pkg string, targets ...string) sourceFile {
		return sourceFile{pkg: pkg, path: pkg + "/file.go", imports: targets}
	}
	cases := []struct {
		name  string
		files []sourceFile
		want  []string
	}{
		{"web imports sqlite", []sourceFile{imports(pkgHandlers, pkgSQLite)},
			[]string{ruleSQLiteOwner + " engine/handlers", ruleWeb + " engine/handlers"}},
		{"web reaches sqlite indirectly", []sourceFile{
			imports(pkgAdminUI, "internal/relayoutbox"), imports("internal/relayoutbox", pkgSQLite)},
			[]string{ruleSQLiteOwner + " internal/relayoutbox", ruleWeb + " internal/adminui"}},
		{"another package imports sqlite", []sourceFile{imports("internal/export", pkgSQLite)},
			[]string{ruleSQLiteOwner + " internal/export"}},
		{"driver outside sqlite", []sourceFile{imports("internal/export", "database/sql", "modernc.org/sqlite/lib")},
			[]string{ruleSQL + " internal/export", ruleSQL + " internal/export"}},
		{"SQL text outside sqlite", []sourceFile{{pkg: "internal/export", path: "x.go", sql: []int{3}}},
			[]string{ruleSQL + " internal/export"}},
		{"core imports an adapter", []sourceFile{imports(pkgModels, pkgMBZ)},
			[]string{ruleCore + " internal/models"}},
		{"core reaches the interface indirectly", []sourceFile{
			imports(pkgIngest, "internal/i18n"), imports("internal/i18n", pkgAdminWeb)},
			[]string{ruleCore + " internal/ingest"}},
		{"domain imports the interface", []sourceFile{imports("internal/catalog", pkgAdminUI, pkgWebUI, pkgHandlers)},
			[]string{ruleDomain + " internal/catalog", ruleDomain + " internal/catalog", ruleDomain + " internal/catalog"}},
		{"sqlite gains a business import", []sourceFile{
			imports(pkgSQLite, "internal/importer"), imports("internal/importer")},
			[]string{ruleSQLiteImport + " internal/db/sqlite"}},
		{"leaf gains an import", []sourceFile{
			imports(pkgImageURL, "internal/catalog", pkgModels), imports("internal/catalog"), imports(pkgModels)},
			[]string{ruleLeaf + " internal/imageurl"}},
		{"allowed imports", []sourceFile{
			imports(pkgEngine, pkgSQLite, pkgWebUI), imports(pkgWebUI, pkgAdminUI),
			imports(pkgSQLite, "database/sql", pkgModels, "github.com/pressly/goose/v3"), imports(pkgModels),
			{pkg: pkgSQLite, path: "q.go", sql: []int{1}}},
			nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, v := range check(c.files) {
				got = append(got, v.rule+" "+v.pkg)
			}
			slices.Sort(c.want)
			slices.Sort(got)
			if !slices.Equal(got, c.want) {
				t.Errorf("violations = %q, want %q", got, c.want)
			}
		})
	}
}

func TestStaleExceptionIsReported(t *testing.T) {
	live := exception{ruleSQL, "a", "database/sql", "still imported"}
	gone := exception{ruleSQL, "b", "database/sql", "import removed"}
	unexpected, stale := compare([]violation{{rule: ruleSQL, pkg: "a", target: "database/sql"}}, []exception{live, gone})
	if len(unexpected) != 0 || !slices.Equal(stale, []exception{gone}) {
		t.Errorf("unexpected = %v, stale = %v", unexpected, stale)
	}
}

func TestLoadModuleReadsOnlyProductionSource(t *testing.T) {
	root := t.TempDir()
	source := "package x\nimport _ \"" + modulePath + "/internal/models\"\n" +
		"var label = \"Delete from library\"\n" +
		"var query = `\n  SELECT id\n  FROM listens`\n" +
		"var change = \"UPDATE listens SET track_id=?\"\n"
	for path, content := range map[string]string{
		"internal/x/x.go":          source,
		"internal/x/x_test.go":     source,
		"internal/x/testdata/y.go": source,
		".hidden/y.go":             source,
		"nested/go.mod":            "module nested\n",
		"nested/y.go":              source,
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := loadModule(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].path != "internal/x/x.go" || files[0].pkg != "internal/x" ||
		!slices.Equal(files[0].imports, []string{pkgModels}) || !slices.Equal(files[0].sql, []int{4, 7}) {
		t.Errorf("files = %+v", files)
	}
}
