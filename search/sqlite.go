package search

import (
	"database/sql"
	"strings"
	"ts_inspector/utils"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

type row struct {
	id        int64
	embedding []float32
	text      string
}

var db *sql.DB

func AddToSqlite(rows []row) error {
	if !utils.SemanticSearchEnableSqlite {
		return nil
	}

	if len(rows) == 0 {
		return nil
	}

	prefix := "INSERT or REPLACE INTO vec_interesting_points(id, embedding, text) VALUES "

	sb := strings.Builder{}
	sb.WriteString(prefix)

	ids := make(map[int64]row)

	args := []any{}
	for i, row := range rows {
		sb.WriteString("(?, ?, ?)")
		vector, err := sqlite_vec.SerializeFloat32(row.embedding)
		if err != nil {
			return err
		}

		ids[row.id] = row
		args = append(args, row.id, vector, row.text)

		if i < len(rows)-1 && i%500 != 0 && i != 0 {
			sb.WriteString(",")
			continue
		}

		sb.WriteString(";")

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
	if !utils.SemanticSearchEnableSqlite || !canUseEmbeddings() {
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
	d, err := sql.Open("sqlite3", ":memory:")
	db = d

	if err != nil {
		return err
	}

	db.Exec(`
		create virtual table vec_interesting_points using vec0(
			id integer primary key,
			embedding float[384],
			text text,
		);
	`)

	if err != nil {
		return err
	}

	return nil
}
