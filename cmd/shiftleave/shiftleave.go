package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jszwec/csvutil"
	"golang.org/x/term"

	_ "github.com/go-sql-driver/mysql"
)

// Replace with your own connection parameters
var server = "auroradb-skillsconn.cluster-ro-cpdlqlocq4io.ap-southeast-2.rds.amazonaws.com"
var port = 6033
var user = "skillsconn_bi"
var password = ""
var dbname = "flg_skillsconn"

type xmlTime time.Time

const xmlTimeFormat = `2006-01-02T15:04:05`

func (ct xmlTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	t := time.Time(ct)
	v := t.Format(xmlTimeFormat)
	return e.EncodeElement(v, start)
}

func (ct *xmlTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	err := d.DecodeElement(&s, &start)
	if err != nil {
		return err
	}
	t, err := time.Parse(xmlTimeFormat, s)
	if err != nil {
		return err
	}
	*ct = xmlTime(t)
	return nil
}

type ShiftHours struct {
	XMLName    xml.Name `xml:"ShiftHours"`
	StartTime  xmlTime  `xml:"StartTime"`
	FinishTime xmlTime  `xml:"FinishTime"`
	Break      *string  `xml:"Break,omitempty"`
	Department string   `xml:"Department"`
	Job        string   `xml:"Job"`
}

type Shift struct {
	XMLName    xml.Name     `xml:"Shift"`
	ShiftHours []ShiftHours `xml:"ShiftHours"`
}

type Allowance struct {
	Type        string  `xml:"Type,attr"`
	AllowanceNo string  `xml:"AllowanceNo"`
	Value       float32 `xml:"Value"`
}

type Leave struct {
	Type     string   `xml:"Type,attr"`
	Hours    float32  `xml:"Hours,attr"`
	FromDate *xmlTime `xml:"FromDate,omitempty"`
	ToDate   *xmlTime `xml:"ToDate,omitempty"`
}

type TimeCard struct {
	XMLName      xml.Name    `xml:"TimeCard"`
	TimeCardNo   string      `xml:"TimeCardNo"`
	EmployeeName string      `xml:"EmployeeName"`
	Shift        []Shift     `xml:"Shift"`
	Allowance    []Allowance `xml:"Allowance,omitempty"`
	TotalHours   float32     `xml:"TotalHours"`
	Leave        []Leave     `xmld:"Leave,omitempty"`
}

type TimeCards struct {
	XMLName   xml.Name   `xml:"TimeCards"`
	Version   string     `xml:"Version,attr"`
	TimeCards []TimeCard `xml:"TimeCard"`
}

type CsvShift struct {
	FullName     string  `csv:"Full Name"`
	Date         string  `csv:"Date"`
	Shift        string  `csv:"Shift"`
	Break        float64 `csv:"Break"`
	Allowance    string  `csv:"Allowance"`
	ActualFirst  float64 `csv:"ActualFirst"`
	ActualSecond float64 `csv:"ActualSecond"`
	IsShort      bool    `csv:"IsShort"`
}

type LeaveData map[string]map[string]float32

const LeaveQuery = `SELECT 
    roster_employee_name AS Staff,
    library_name AS Field,
    attribute_value AS Value
FROM
    roster_employee e,
    globe_object_has_globe_shape o,
    globe_shape s,
    globe_library l,
    globe_attribute t
WHERE
    e.roster_employee_globeuid = o.object_uid
        AND o.shape_id = s.shape_id
        AND library_shape_id = s.shape_id
        AND l.library_id = t.library_id
        AND e.roster_employee_globeuid = t.object_uid
        AND o.ohs_id = t.ohs_id
        AND roster_employee_deleted = 'no'
        AND o.ohs_deleted = 'no'
        AND s.shape_name like 'Leave%'
        and length(l.library_name) = 3
        AND attribute_islatest = 'yes'
group BY o.object_uid , s.shape_id , l.library_id, attribute_revision
`

func credentials() (string, error) {
	fmt.Print("Enter Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	return strings.TrimSpace(password), nil
}

func main() {
	log.Printf("started\n")
	if len(os.Args) < 2 {
		fmt.Println("usage: shiftleave <timesheet>.wtc")
		fmt.Println("   that will produce <timesheet>_out.xtc and <timesheet_out>.csv")
		return
	}

	var err error
	if password == "" {
		password, err = credentials()
		if err != nil {
			log.Fatal(fmt.Sprintf("Error getting password: %v", err))
		}
	}

	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	fileNameOut := fileName[:len(fileName)-len(ext)] + "_out" + ext
	csvFileNameOut := fileName[:len(fileName)-len(ext)] + "_out" + ".csv"

	leaveData, err := readLeaveData()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading data from DB: %v", err))
	}
	log.Printf("loaded %d staff from DB", len(leaveData))

	timeCards, err := readCards(fileName)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't parse file: %s: %v", fileName, err))
	}

	csvShifts, err := processAllowances(timeCards)
	if err != nil {
		log.Fatal(fmt.Sprintf("process allowances: %v", err))
	}

	err = processCards(timeCards)
	if err != nil {
		log.Fatal(fmt.Sprintf("process cards: %v", err))
	}

	err = processLeaves(timeCards, leaveData)
	if err != nil {
		log.Fatal(fmt.Sprintf("process leaves: %v", err))
	}

	err = writeCsvShifts(csvShifts, csvFileNameOut)
	if err != nil {
		log.Fatal(fmt.Sprintf("writting csvs: %v", err))
	}

	err = writeCards(timeCards, fileNameOut)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't write file: %s: %v", fileNameOut, err))
	}
}

func readLeaveData() (LeaveData, error) {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, server, port, dbname)
	db, err := sql.Open("mysql", connString)

	if err != nil {
		log.Fatal("Error creating connection pool: " + err.Error())
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	rows, err := db.Query(LeaveQuery)
	if err != nil {
		return nil, err
	}
	var name string
	var field string
	var value float32
	leaveData := make(LeaveData)
	for rows.Next() {
		err := rows.Scan(&name, &field, &value)
		if err != nil {
			return nil, err
		}
		staff, ok := leaveData[name]
		if !ok {
			staff = make(map[string]float32)
			leaveData[name] = staff
		}
		staff[field] = value
	}
	return leaveData, nil
}

func readCards(fileName string) (*TimeCards, error) {
	buff, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	timeCards := &TimeCards{}
	err = xml.Unmarshal(buff, &timeCards)
	if err != nil {
		return nil, err
	}

	log.Printf("read: %d time cards", len(timeCards.TimeCards))
	return timeCards, nil
}

func writeCards(timeCards *TimeCards, fileName string) error {
	xmlFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	xmlFile.WriteString(xml.Header)
	encoder := xml.NewEncoder(xmlFile)
	encoder.Indent("", "\t")
	err = encoder.Encode(timeCards)
	if err != nil {
		return err
	}
	log.Printf("done, for: %d time cards", len(timeCards.TimeCards))
	return nil
}

func processCards(timeCards *TimeCards) error {
	for i, tc := range timeCards.TimeCards {
		for j, shift := range tc.Shift {
			arr := make([]ShiftHours, 0)
			for _, sh := range shift.ShiftHours {
				if sh.Break != nil && *sh.Break != "00:00" {
					start := time.Time(sh.StartTime)
					finish := time.Time(sh.FinishTime)
					shiftDuration := finish.Sub(start)
					hs, err := strconv.ParseUint((*sh.Break)[:2], 10, 32)
					if err != nil {
						return fmt.Errorf("can't parse Break: %s", *sh.Break)
					}
					mi, err := strconv.ParseUint((*sh.Break)[3:], 10, 32)
					if err != nil {
						return fmt.Errorf("can't parse Break: %s", *sh.Break)
					}
					breakDuration := time.Duration(int64(hs*60+mi) * 60 * 1000 * 1000 * 1000)
					shiftDuration -= breakDuration
					sh1 := ShiftHours{
						StartTime:  sh.StartTime,
						FinishTime: xmlTime(time.Time(sh.StartTime).Add(shiftDuration / 2)),
						Department: sh.Department,
						Job:        sh.Job,
					}
					sh2 := ShiftHours{
						StartTime:  xmlTime(time.Time(sh.FinishTime).Add(-shiftDuration / 2)),
						FinishTime: sh.FinishTime,
						Department: sh.Department,
						Job:        sh.Job,
					}
					arr = append(arr, sh1)
					arr = append(arr, sh2)
				} else {
					sh.Break = nil
					arr = append(arr, sh)
				}
			}
			timeCards.TimeCards[i].Shift[j].ShiftHours = arr
		}
		sort.Slice(tc.Shift, func(i1 int, i2 int) bool {
			s1 := tc.Shift[i1].ShiftHours[0]
			s2 := tc.Shift[i2].ShiftHours[0]
			return time.Time(s1.StartTime).Before(time.Time(s2.StartTime))
		})

	}
	return nil
}

func leaveDataHasStaff(leaveData LeaveData, name string) bool {
	_, ok := leaveData[name]
	return ok
}

func leaveDataHours(leaveData LeaveData, name string, fromDate xmlTime, dayOffset int) (float32, error) {
	date := time.Time(fromDate).Add(time.Hour * time.Duration(24*dayOffset))
	weekDay := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}[date.Weekday()%6]
	hours, ok := leaveData[name][weekDay]
	if !ok {
		return 0, nil
	}
	return hours, nil
}

func processLeaves(timeCards *TimeCards, leaveData LeaveData) error {
	dayOffset := 0
	currentType := ""
	for i, tc := range timeCards.TimeCards {
		isStaff := leaveDataHasStaff(leaveData, tc.EmployeeName)
		for j, leave := range tc.Leave {
			switch leave.Type {
			case "Time in Lieu":
				if isStaff {
					timeCards.TimeCards[i].Leave[j].Type = "Accrued Days"
				} else {
					timeCards.TimeCards[i].Leave[j].Type = "ADO"
				}
			case "Annual", "Sick", "Long Service", "Compassionate":
				if isStaff && leave.Hours == 7.6 {
					if currentType != leave.Type {
						dayOffset = 0
					}
					currentType = leave.Type
					hours, err := leaveDataHours(leaveData, tc.EmployeeName, *leave.FromDate, dayOffset)
					if err != nil {
						return err
					}
					timeCards.TimeCards[i].Leave[j].Hours = hours
					dayOffset++
				}
			}
			timeCards.TimeCards[i].Leave[j].FromDate = nil
			timeCards.TimeCards[i].Leave[j].ToDate = nil
		}
		dayOffset = 0
		currentType = ""
	}
	return nil
}

func processAllowances(timeCards *TimeCards) ([]CsvShift, error) {
	csvShifts := make([]CsvShift, 0)
	for i, tc := range timeCards.TimeCards {
		allowances := make([]Allowance, 0)
		cntAllowances := 0
		continuesShift := float64(0)
		cntBreaks := 0
		shiftNdx := 0
		var previousShift *ShiftHours
		isOvernight := false
		for i, shift := range tc.Shift {
			for j, sh := range shift.ShiftHours {
				// is new day
				if previousShift != nil && truncateToDay(sh.StartTime) != truncateToDay(previousShift.StartTime) {
					shiftNdx = 0
				}

				if shiftNdx == 0 {
					cntAllowances = 0
					continuesShift = 0
					previousShift = nil
					cntBreaks = 0
				}
				// instead of isOvernight check that it is not started at midnight:
				//     isOvernight = truncateToDay(sh.StartTime) != truncateToDay(sh.FinishTime)
				isOvernight = truncateToDay(sh.StartTime) == time.Time(sh.StartTime)

				start := time.Time(sh.StartTime)
				finish := time.Time(sh.FinishTime)
				shiftDuration := finish.Sub(start).Hours()

				breakDuration := float64(0)
				if previousShift != nil {
					breakDuration = time.Time(sh.StartTime).Sub(time.Time(previousShift.FinishTime)).Hours()
					if breakDuration < 0 {
						msg := fmt.Sprintf(
							"prev: StartTime: %s FinishTime: %s sh: StartTime: %s FinishTime: %s",
							time.Time(previousShift.StartTime).Format("2006-01-02 15:04"),
							time.Time(previousShift.FinishTime).Format("2006-01-02 15:04"),
							time.Time(sh.StartTime).Format("2006-01-02 15:04"),
							time.Time(sh.FinishTime).Format("2006-01-02 15:04"),
						)
						log.Println(msg)
					}
				}

				if shiftNdx == 0 {
					breakDuration = 0
				}

				strAllowance := ""
				actualFirst := float64(0)
				actualSecond := float64(0)
				// ignore overnight shift (starts at 0:00)
				if !isOvernight {
					if breakDuration > 0 {
						cntBreaks++
					}
					if cntAllowances == 0 &&
						((breakDuration > 1 || (breakDuration > 0 && (continuesShift > 5 || shiftDuration > 5))) ||
							(breakDuration > 0 && cntBreaks > 1)) {
						strAllowance = "First"
						allowances = append(allowances, Allowance{
							Type:        "Unit",
							AllowanceNo: "First Broken",
							Value:       float32(breakDuration),
						})
						actualFirst = float64(breakDuration)
						cntAllowances++
					} else if cntAllowances == 1 && breakDuration > 0 {
						strAllowance = "Second"
						allowances = allowances[:len(allowances)-1]
						allowances = append(allowances, Allowance{
							Type:        "Unit",
							AllowanceNo: "Second Broken",
							Value:       float32(breakDuration),
						})
						actualSecond = float64(breakDuration)
						cntAllowances++
					}
				}

				isShort := false
				if shiftDuration < 2 {
					isJoined := false
					var prevSh *ShiftHours
					if j > 0 {
						prevSh = &shift.ShiftHours[j-1]
					}
					if j == 0 && i > 0 {
						prevSh = &tc.Shift[i-1].ShiftHours[len(tc.Shift[i-1].ShiftHours)-1]
					}
					if prevSh != nil && prevSh.FinishTime == sh.StartTime {
						isJoined = true
					}
					var nextSh *ShiftHours
					if j < len(shift.ShiftHours)-1 {
						nextSh = &shift.ShiftHours[j+1]
					}
					if j == len(shift.ShiftHours)-1 && i < len(tc.Shift)-1 {
						nextSh = &tc.Shift[i+1].ShiftHours[0]
					}
					if nextSh != nil && nextSh.StartTime == sh.FinishTime {
						isJoined = true
					}
					if !isJoined {
						isShort = true
					}
				}

				csvShifts = append(csvShifts, CsvShift{
					FullName:     tc.EmployeeName,
					Date:         printDate(sh.StartTime),
					Shift:        printStartFinish(sh),
					Break:        breakDuration,
					Allowance:    strAllowance,
					ActualFirst:  actualFirst,
					ActualSecond: actualSecond,
					IsShort:      isShort,
				})

				shiftNdx++
				continuesShift += shiftDuration
				previousShift = &shift.ShiftHours[j]
			}
		}
		totalFirstCnt := 0
		totalSecondCnt := 0
		for _, a := range allowances {
			if a.AllowanceNo == "First Broken" {
				totalFirstCnt++
			}
			if a.AllowanceNo == "Second Broken" {
				totalSecondCnt++
			}
		}
		if totalFirstCnt > 0 {
			timeCards.TimeCards[i].Allowance = append(timeCards.TimeCards[i].Allowance, Allowance{
				Type:        "Unit",
				AllowanceNo: "First Broken",
				Value:       float32(totalFirstCnt),
			})
		}
		if totalSecondCnt > 0 {
			timeCards.TimeCards[i].Allowance = append(timeCards.TimeCards[i].Allowance, Allowance{
				Type:        "Unit",
				AllowanceNo: "Second Broken",
				Value:       float32(totalSecondCnt),
			})
		}
	}
	return csvShifts, nil
}

func truncateToDay(x xmlTime) time.Time {
	t := time.Time(x)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func printDate(x xmlTime) string {
	return time.Time(x).Format("2006-01-02")
}
func printStartFinish(sh ShiftHours) string {
	return time.Time(sh.StartTime).Format("15:04") + " - " + time.Time(sh.FinishTime).Format("15:04")
}

func writeCsvShifts(csvShifts []CsvShift, filename string) error {
	b, err := csvutil.Marshal(csvShifts)
	if err != nil {
		fmt.Println("error:", err)
	}
	bb := []byte("{}\n")
	bb = append(bb, b...)
	return ioutil.WriteFile(filename, bb, 0644)
}
