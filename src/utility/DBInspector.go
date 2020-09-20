package utility

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
)

type DBInspector struct {
	dataSource string
	connection *sql.DB
}

func MapRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columnTypes, e := rows.ColumnTypes()
	if e != nil {
		return nil, e
	}
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		columnPointers := make([]interface{}, len(columnTypes))
		for i := range columnPointers {
			columnPointers[i] = reflect.New(columnTypes[i].ScanType()).Interface()
		}
		if e := rows.Scan(columnPointers...); e != nil {
			return nil, e
		}
		temp := make(map[string]interface{})
		for i := range columnPointers {
			temp[columnTypes[i].Name()] = reflect.ValueOf(columnPointers[i]).Elem().Interface()
		}
		result = append(result, temp)
	}
	return result, nil
}

func NewDBInspector(dataSource string) (*DBInspector, error) {
	result := new(DBInspector)
	result.dataSource = dataSource
	connection, e := sql.Open("mysql", dataSource)
	if e != nil {
		return nil, e
	}
	result.connection = connection
	return result, nil
}

func DumpColumnsInfo(rows *sql.Rows) error {
	types, e := rows.ColumnTypes()
	if e != nil {
		return e
	}

	for i := range types {
		fmt.Println(types[i].Name() + " " + types[i].ScanType().Name())
	}
	return nil
}

// Execute arbitrary query
func (inspector *DBInspector) Query(statement string) ([]map[string]interface{}, error) {
	rows, e := inspector.connection.Query(statement)
	if e != nil {
		return nil, e
	}
	return MapRows(rows)
}

func (inspector *DBInspector) PrintProcessList() error {
	statement, e := inspector.connection.Prepare(`select * from information_schema.PROCESSLIST`)
	if e != nil {
		return e
	}
	rows, e := statement.Query()
	if e != nil {
		return e
	}
	result, e := MapRows(rows)
	if e != nil {
		return e
	}

	for i := range result {
		id := result[i]["ID"].(uint64)
		host := string(result[i]["HOST"].(sql.RawBytes))
		command := string(result[i]["COMMAND"].(sql.RawBytes))
		info := string(result[i]["INFO"].(sql.RawBytes))
		fmt.Printf("%d\t%s\t%s\t%s\n", id, host, command, info)
	}
	return nil
}

// show index from `database`.`table`
func (inspector *DBInspector) ShowIndex(database, table string) error {
	statement, e := inspector.connection.Prepare(`
		select *
        from information_schema.statistics
        where table_schema = ? and table_name = ?`)
	if e != nil {
		return e
	}
	rows, e := statement.Query(database, table)
	if e != nil {
		return e
	}
	result, e := MapRows(rows)
	for i := range result {
		temp := map[string]interface{}{
			"TABLE_CATALOG": string(result[i]["TABLE_CATALOG"].(sql.RawBytes)),
			"NON_UNIQUE":    result[i]["NON_UNIQUE"].(int64),
			"INDEX_SCHEMA":  string(result[i]["INDEX_SCHEMA"].(sql.RawBytes)),
			"INDEX_NAME":    string(result[i]["INDEX_NAME"].(sql.RawBytes)),
			"INDEX_TYPE":    string(result[i]["INDEX_TYPE"].(sql.RawBytes)),
			"COLUMN_NAME":   string(result[i]["COLUMN_NAME"].(sql.RawBytes)),
			"COLLATION":     string(result[i]["COLLATION"].(sql.RawBytes)),
			"CARDINALITY":   result[i]["CARDINALITY"].(sql.NullInt64).Int64,
		}
		tempBytes, e := json.Marshal(temp)
		if e != nil {
			return e
		}
		fmt.Println(string(tempBytes))
	}
	return nil
}

func (inspector *DBInspector) Close() error {
	return inspector.connection.Close()
}
