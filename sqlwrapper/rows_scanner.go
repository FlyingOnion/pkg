package sqlwrapper

import (
	"database/sql"
	"reflect"
)

type singleRowScanner struct {
	dest []interface{}
}

func (ds *singleRowScanner) ScanFrom(r *sql.Rows) (err error) {
	r.Next()
	err = r.Scan(ds.dest...)
	return
}

// SingleRowScanner wraps the destination of scan to implement RowsScanner.
// Single use.
func SingleRowScanner(dest ...interface{}) RowsScanner {
	return &singleRowScanner{dest}
}

// RowsScanner provide an iterator-like function in the query.
// You can use it to scan the values to your targets.
// You don't need to call r.Close after scan.
//
//	// use SingleRowScanner to wrap your destinations
//	// SingleRowScanner is single-used, which means that it only works fine when result contains only 1 row
//	var id int64
//	var name string
//	scanner := SingleRowScanner(&id, &name)
//	db.RawQuery(ctx, "select id, name from user limit 1", scanner)
//
//	// single-row count aggregation
//	var count int
//	scanner := SingleRowScanner(&count)
//	db.RawQuery(ctx, "select count(*) from user", scanner)
//
//	// you could implement your own RowsScanner in the cases that the query result contains multiple rows. Use ScanFn to wrap the function if needed.
//	users := []*User{}
//	fn := func (r *sql.Rows) (err error) {
//	    // prescan logics, like initializing variables
//	    var id int64
//	    var name string
//	    // do the scan loop
//	    for r.Next() {
//	        err = r.Scan(&id, &name)
//	        if err != nil {
//	            return
//	        }
//	        // append {id, name} to slice, or whatever.
//	        users = append(users, &User{id, name})
//	    }
//	    // postscan logics
//	    fmt.Println(users)
//	    return
//	}
//	db.RawQuery(ctx, "select id, name from user", ScanFn(fn))
type RowsScanner interface {
	ScanFrom(r *sql.Rows) error
}

// ScanFn is a wrapper of user-defined scan function
type ScanFn func(r *sql.Rows) error

func (fn ScanFn) ScanFrom(r *sql.Rows) error {
	return fn(r)
}

func StructScanner(
	entity interface{},
	meta *structMeta,
	converter ValueConverter,
	onNull Strategy,
	unused map[string]interface{},
	found *bool,
) RowsScanner {
	return ScanFn(func(rows *sql.Rows) (err error) {
		// get columns
		cols, err := rows.Columns()
		if err != nil {
			return
		}

		*found = rows.Next()
		if !*found {
			return
		}
		v := reflect.ValueOf(entity).Elem()

		nColumns := len(cols)
		ifacePtrs := make([]interface{}, 0, nColumns)

		for i := 0; i < nColumns; i++ {
			ifacePtrs = append(ifacePtrs, new(interface{}))
		}
		// we use pointers of empty interfaces to be the target of scan
		_ = rows.Scan(ifacePtrs...)
		// after scan all values are stored in ifaces respectively

		for i := 0; i < nColumns; i++ {
			src := *(ifacePtrs[i].(*interface{}))
			fm, ok := meta.columnFieldMap[cols[i]]
			if !ok {
				if unused != nil {
					unused[cols[i]] = src
				}
				continue
			}
			dptr := v.Field(fm.index).Addr().Interface()
			err = convertValue(dptr, src, converter, onNull)
			if err != nil {
				return
			}
		}
		return
	})
}

func SliceScanner(
	slice interface{},
	meta *structMeta,
	converter ValueConverter,
	onNull Strategy,
	elemType reflect.Type,
	isPointer bool,
) RowsScanner {
	return ScanFn(func(rows *sql.Rows) error {
		cols, err := rows.Columns()
		if err != nil {
			// TODO: add a warning log
			return err
		}
		nColumns := len(cols)

		ifacePtrs := make([]interface{}, 0, nColumns)
		for i := 0; i < nColumns; i++ {
			ifacePtrs = append(ifacePtrs, new(interface{}))
		}

		newElems := []reflect.Value{}
		for rows.Next() {
			_ = rows.Scan(ifacePtrs...)
			vPtr := reflect.New(elemType)
			v := vPtr.Elem()
			for i := 0; i < nColumns; i++ {
				fm, ok := meta.columnFieldMap[cols[i]]
				if !ok {
					continue
				}
				err = convertValue(
					v.Field(fm.index).Addr().Interface(),
					*(ifacePtrs[i].(*interface{})),
					converter, onNull,
				)
				if err != nil {
					return err
				}
			}
			if isPointer {
				newElems = append(newElems, vPtr)
				continue
			}
			newElems = append(newElems, v)
		}
		vSlice := reflect.ValueOf(slice).Elem()
		vSlice.Set(reflect.Append(vSlice, newElems...))
		return nil
	})
}
