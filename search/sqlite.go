package search

import (
	"database/sql"
	"slices"
	"strings"
	"ts_inspector/config"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

type row struct {
	id        int64
	embedding []float32
	text      string
}

var db *sql.DB

func DeleteInterestingFromUri(rootPath string) error {
	_, err := db.Exec("DELETE FROM vec_interesting_points WHERE rootPath = ?;", rootPath)
	if err != nil {
		return err
	}

	return nil
}

func AddToSqlite(table string, rows []row, columns []string, ignoreConflicts []string, appendArgs func([]any, row) []any) error {
	if !config.GetConfig().SemanticSearch.EnableSqlite {
		return nil
	}

	if len(rows) == 0 {
		return nil
	}

	prefix := "INSERT or IGNORE INTO " + table + "(" + strings.Join(columns, ", ") + ") VALUES "

	suffix := ""
	if len(ignoreConflicts) > 0 {
		suffix = " ON CONFLICT(" + strings.Join(ignoreConflicts, ", ") + ") DO NOTHING;"
	}

	sb := strings.Builder{}
	sb.WriteString(prefix)

	ids := make(map[int64]row)

	placeholders := strings.Join(slices.Repeat([]string{"?"}, len(columns)), ", ")

	args := []any{}
	for i, row := range rows {
		sb.WriteString("(" + placeholders + ")")

		ids[row.id] = row
		args = appendArgs(args, row)

		if i < len(rows)-1 && i%500 != 0 && i != 0 {
			sb.WriteString(",")
			continue
		}

		sb.WriteString(suffix)

		insertStatement, err := db.Prepare(sb.String())
		if err != nil {
			return err
		}

		sb.Reset()
		sb.WriteString(prefix)

		_, err = insertStatement.Exec(args...)
		if err != nil {
			return err
		}

		args = []any{}
	}

	return nil
}

func SearchSqlite(queryText string, resultsCount int64) ([]Result, error) {
	if !config.GetConfig().SemanticSearch.EnableSqlite || !canUseEmbeddings() {
		return []Result{}, nil
	}

	queryVector, err := sqlite_vec.SerializeFloat32(GetEmbedding(queryText))
	if err != nil {
		return []Result{}, err
	}

	rows, err := db.Query(`
		select id, text, distance
		from vec_interesting_points
		where embedding match ? and k = ?
	`, queryVector, resultsCount)

	if err != nil {
		return []Result{}, err
	}

	results := make([]Result, resultsCount)

	i := 0
	for rows.Next() {
		var id int64
		var text string
		var distance float64

		if e := rows.Scan(&id, &text, &distance); e != nil {
			return nil, e
		}

		results[i] = Result{float32(1 - distance + SortOrderEmbedding), id, "sqlite"}

		i++
	}

	return results, nil
}

func initSqlite() error {
	sqlite_vec.Auto()
	d, err := sql.Open("sqlite3", "./sqlite.sqlite")
	db = d

	if err != nil {
		return err
	}

	_, err = db.Exec(`
		create virtual table if not exists vec_interesting_points using vec0(
			id integer,
			embedding float[384],
			rootPath text,
			text text,
		);

		create table if not exists vec_cache (
			embedding blob,
			text text unique
		);

	`)

	if err != nil {
		return err
	}

	return nil
}
