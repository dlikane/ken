package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Replace with your own connection parameters
var server = "auroradb-skillsconn.cluster-ro-cpdlqlocq4io.ap-southeast-2.rds.amazonaws.com"
var port = 6033
var user = "skillsconn_bi"
var password = "mbq.HPH2uvp4adu.dnk"
var dbname = "flg_skillsconn"

var baseQuery = `SELECT 
roster_position_name AS Participant,
roster_shift_start,
roster_shift_end,
roster_employee_name AS Staff,
site_name,
IFNULL(replace (replace(r.roster_shift_note, '\n', ' '), '\r', ' '), '') AS Shift_note
FROM
roster_shift r,
roster_position p,
roster_employee e,
roster_shift_has_employee h,
globe_site s
WHERE
DATE(roster_shift_start) >= NOW()
	AND DATE(roster_shift_start) < (NOW() + INTERVAL 1 MONTH)
	AND r.roster_position_id = p.roster_position_id
	AND h.roster_shift_id = r.roster_shift_id
	AND h.roster_employee_id = e.roster_employee_id
	AND r.site_id = s.site_id
	AND roster_shift_deleted = 'no'
ORDER BY site_name , roster_position_name , roster_shift_start
LIMIT 111111
`

func main() {
	log.Println("rosternext started")

	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, server, port, dbname)
	db, err := sql.Open("mysql", connString)

	if err != nil {
		log.Fatal("Error creating connection pool: " + err.Error())
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = processQuery(db)
	if err != nil {
		log.Fatal("Error query DB" + err.Error())
	}

	log.Println("rosternext ended")
}

func processQuery(db *sql.DB) error {
	query := baseQuery
	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtr := make([]interface{}, count)
	valueStr := make([]string, count)

	for i := range columns {
		valuePtr[i] = &values[i]
	}
	fmt.Println(strings.Join(columns, ","))
	for rows.Next() {
		rows.Scan(valuePtr...)
		for i := range columns {
			val := values[i]
			b, ok := val.([]byte)
			var s string
			if ok {
				s = string(b)
			} else {
				s = fmt.Sprintf("%v", val)
			}
			valueStr[i] = s
		}
		fmt.Println(strings.Join(valueStr, ","))
	}
	return nil
}
