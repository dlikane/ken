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
    bb.username AS 'Staff',
    DATE_FORMAT(bb.roster_shift_start, '%a, %d %b %Y') AS 'Date',
        TIME_FORMAT(bb.ts_start_time, '%h:%i %p') AS 'Clock ON',
    TIME_FORMAT(bb.ts_end_time, '%h:%i %p') AS 'Clock Off',
    ROUND((HOUR(bb.ts_spenttime) + (MINUTE(bb.ts_spenttime) / 60)) - (HOUR(bb.ts_breaktime) + (MINUTE(bb.ts_breaktime) / 60)),
            4) AS Worked,
    bb.Billed,
    TIME(bb.roster_shift_start) AS 'Shift Start tm',
    DATE(bb.roster_shift_end) AS 'Shift Start dt',
    TIME(bb.roster_shift_end) AS 'Shift End tm',
    IFNULL(bb.rs_billing_ratio_a, '') AS A,
    IFNULL(bb.rs_billing_ratio_b, '') AS B,
    bb.roster_position_name AS 'Participant',
    IFNULL(bb.ts_notes, '') AS ts_notes,
    IFNULL(bb.roster_shift_note, '') AS shift_note,
    IFNULL(bb.ts_task, '') AS ts_task,
    bb.site_name
FROM
    (SELECT 
        SUM(b.rs_billing_quantity / ifnull( b.rs_billing_ratio_b,1)) AS Billed,
            u.username,
            t.ts_spenttime,
            t.ts_breaktime,
            t.ts_start_date,
            t.ts_start_time,
            t.ts_end_time,
            a.roster_shift_start,
            a.roster_shift_end,
            b.rs_billing_ratio_a,
            b.rs_billing_ratio_b,
            p.roster_position_name,
            a.roster_shift_cancelled,
            a.roster_shift_note,
            t.ts_task,
            s.site_name,
            t.ts_notes
    FROM
        roster_shift a, roster_position p, globe_site s, opt_timesheet t, opt_user u, roster_shift_billable b, roster_shift_has_employee re, roster_employee e
    WHERE
        b.roster_shift_id = a.roster_shift_id
            AND u.idopt_user = t.idopt_user
            AND p.roster_position_id = a.roster_position_id
            AND re.roster_shift_id = a.roster_shift_id
            AND re.roster_employee_id = e.roster_employee_id
            AND e.roster_employee_idoptuser = t.idopt_user
            AND s.site_id = t.site_id
            AND ((DATE(a.roster_shift_start) = t.ts_start_date
            AND DATE(a.roster_shift_end) = t.ts_end_date
            AND TIME(a.roster_shift_end) <= t.ts_end_time
            AND ts_start_date = ts_end_date)
            OR (ts_start_date < ts_end_date
            AND DATE(a.roster_shift_start) = t.ts_end_date
           AND DATE(a.roster_shift_end) = t.ts_end_date
           ))
            AND TIME(a.roster_shift_start) >= t.ts_start_time
            AND ts_status in ('Approved','Processed')
            AND b.rs_billing_flag LIKE 'f%'
            AND b.rs_billing_code NOT LIKE '87%'
            AND ROUND((HOUR(t.ts_spenttime) + (MINUTE(t.ts_spenttime) / 60)) - (HOUR(t.ts_breaktime) + (MINUTE(t.ts_breaktime) / 60)), 4) <> b.rs_billing_quantity
            AND (roster_shift_cancelled < 10
            OR roster_shift_cancelled > 60)
            AND s.site_name IN ('In Home Support' , 'Child and Youth', 'Our Connections', 'Community Programs')
			AND YEAR(roster_shift_start) * 100 + MONTH(roster_shift_start) = ?
            AND a.roster_shift_deleted = 'no'
            AND b.rs_billing_deleted = 'no'
    GROUP BY u.username , t.ts_start_date , t.ts_start_time) bb
WHERE
    ROUND((HOUR(bb.ts_spenttime) + (MINUTE(bb.ts_spenttime) / 60)) - (HOUR(bb.ts_breaktime) + (MINUTE(bb.ts_breaktime) / 60)),
            4) <> ROUND(bb.Billed, 4)
ORDER BY bb.username , bb.ts_start_date , bb.roster_shift_start
LIMIT 10000`

func main() {
	log.Println("timediff started")

	if len(os.Args) < 2 {
		fmt.Println("usage: timediff YYYYMM")
		fmt.Println("   YYYY - year (for instance 2021)")
		fmt.Println("   MM   - month (for instance 02)")
		return
	}

	from, err := strconv.Atoi(os.Args[1])
	if err != nil || from < 200000 || from > 300000 {
		log.Fatalf("Can't parse from: %s", os.Args[1])
	}

	log.Printf("Running for: %d\n", from)

	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, server, port, dbname)
	db, err := sql.Open("mysql", connString)

	if err != nil {
		log.Fatal("Error creating connection pool: " + err.Error())
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = processQuery(db, from)
	if err != nil {
		log.Fatal("Error query DB" + err.Error())
	}

	log.Println("timediff ended")
}

func processQuery(db *sql.DB, from int) error {
	rows, err := db.Query(baseQuery, from)
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
			if strings.Contains(valueStr[i], ",") {
				valueStr[i] = `"` + valueStr[i] + `"`
			}
		}
		fmt.Println(strings.Join(valueStr, ","))
	}
	return nil
}
