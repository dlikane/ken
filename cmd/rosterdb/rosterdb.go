package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
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
    e.roster_employee_id,
    p.roster_position_name,
    e.roster_employee_name,
    s.site_name,
    a.roster_shift_start,
    a.roster_shift_end,
    a.roster_shift_cancelled,
    a.roster_shift_note,
    a.roster_shift_id
FROM
    roster_shift a,
    roster_position p,
    globe_site s,
    roster_employee e,
    roster_shift_has_employee se
WHERE
    p.roster_position_id = a.roster_position_id
        AND s.site_id = a.site_id
        AND e.roster_employee_id = se.roster_employee_id
        AND se.roster_shift_id = a.roster_shift_id
        AND s.site_name = 'In Home Support'
		AND YEAR(a.roster_shift_start) * 100 + MONTH(a.roster_shift_start) BETWEEN ? AND ?
		AND p.roster_position_enabled = 'yes'
        AND roster_employee_enabled = 'yes'
        AND roster_shift_deleted = 'no'
        AND roster_employee_deleted = 'no'
        AND roster_position_deleted = 'no'
`

func main() {
	log.Println("rosterdb started")

	if len(os.Args) < 3 {
		fmt.Println("usage: rosterdb YYYYMM_from YYYYMM_to")
		fmt.Println("   YYYY - year (for instance 2021)")
		fmt.Println("   MM   - month (for instance 02)")
		return
	}

	from, err := strconv.Atoi(os.Args[1])
	if err != nil || from < 200000 || from > 300000 {
		log.Fatalf("Can't parse from: %s", os.Args[1])
	}

	to, err := strconv.Atoi(os.Args[2])
	if err != nil || to < 200000 || to > 300000 {
		log.Fatalf("Can't parse to: %s", os.Args[2])
	}
	if to < from {
		log.Fatalf("From shoud be less than to: %d %d", from, to)
	}

	log.Printf("Running for: %d %d\n", from, to)

	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, server, port, dbname)
	db, err := sql.Open("mysql", connString)

	if err != nil {
		log.Fatal("Error creating connection pool: " + err.Error())
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = processQuery(db, from, to)
	if err != nil {
		log.Fatal("Error query DB" + err.Error())
	}

	log.Println("rosterdb ended")
}

func processQuery(db *sql.DB, from int, to int) error {
	query := `SELECT
t1.roster_employee_name,
t1.roster_shift_start,
t1.roster_shift_end,
t1.roster_shift_cancelled,
t1.roster_shift_note,
t2.roster_shift_start,
t2.roster_shift_end,
t2.roster_shift_cancelled,
t2.roster_shift_note` + " FROM (" + baseQuery + ") t1, (" + baseQuery + ") t2" + `
WHERE t1.roster_employee_id = t2.roster_employee_id
AND t1.roster_shift_id <> t2.roster_shift_id
AND (
	t1.roster_shift_start > t2.roster_shift_start AND t1.roster_shift_start < t2.roster_shift_end
	OR t1.roster_shift_end > t2.roster_shift_start AND t1.roster_shift_end < t2.roster_shift_end
)`

	rows, err := db.Query(query, from, to, from, to)
	if err != nil {
		return err
	}

	columns, _ := rows.Columns()
	count := len(columns)
	values := make([]interface{}, count)
	valuePtr := make([]interface{}, count)
	valueStr := make([]string, count)

	for i, _ := range columns {
		valuePtr[i] = &values[i]
	}
	fmt.Println(strings.Join(columns, ","))
	for rows.Next() {
		rows.Scan(valuePtr...)
		for i, _ := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				valueStr[i] = string(b)
			} else {
				valueStr[i] = fmt.Sprintf("%v", val)
			}
		}
		fmt.Println(strings.Join(valueStr, ","))
	}
	return nil
}
