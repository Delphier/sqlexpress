package godac

import (
	"database/sql"
	"godac/sqlbuilder"
)

// DB is a interface of *sql.DB and *sql.Tx.
type DB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Map is a shortcut for map[string]interface{}, represents a database record.
type Map map[string]interface{}

// State is dataset state.
type State byte

// States enum.
const (
	StateUnknown State = iota
	StateInsert
	StateUpdate
	StateDelete
)

// Context contains the environment information on Insert/Update/Delete.
type Context struct {
	State   State
	DB      DB
	DataSet DataSet
	Table   *Table
	Record  Map
	Field   Field
}

// Result is an extension of sql.Result.
type Result interface {
	Record(refresh bool) (Map, error) // Get last Insert/Update record, set refresh is true to requery from database.
	sql.Result
}

type result struct {
	context   Context
	sqlResult sql.Result
}

// NewResult create result
func NewResult(c Context, r sql.Result) Result {
	return result{c, r}
}

func (r result) Record(refresh bool) (Map, error) {
	if !refresh {
		return r.context.Record, nil
	}
	var record = Map{}
	for k, v := range r.context.Record {
		record[k] = v
	}
	table := r.context.Table
	if r.context.State == StateInsert && table.autoInc >= 0 && table.Fields[table.autoInc].PrimaryKey {
		id, err := r.sqlResult.LastInsertId()
		if err != nil {
			return nil, err
		}
		record[table.keys[table.autoInc]] = id
	}
	query, args, err := table.WherePrimaryKey(false, false, record)
	if err != nil {
		return nil, err
	}
	maps, err := r.context.DataSet.Select(r.context.DB, sqlbuilder.Select().Where(query), args...)
	if err != nil || len(maps) == 0 {
		return nil, err
	}
	for k, v := range maps[0] {
		record[k] = v
	}
	return record, nil
}

func (r result) LastInsertId() (int64, error) {
	return r.sqlResult.LastInsertId()
}
func (r result) RowsAffected() (int64, error) {
	return r.sqlResult.RowsAffected()
}

// ActionFunc is customize func for Insert/Update/Delete
type ActionFunc func(Context) (Result, error)

// DataSet represents Table or Query.
type DataSet interface {
	Select(db DB, selector sqlbuilder.Selector, args ...interface{}) ([]Map, error)
	Insert(db DB, record Map) (Result, error)
	Update(db DB, record Map) (Result, error)
	Delete(db DB, record Map) (Result, error)
}
