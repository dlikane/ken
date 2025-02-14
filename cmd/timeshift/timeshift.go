package main

import (
	"crypto/tls"
	"crypto/x509"
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

	"github.com/go-sql-driver/mysql"
)

// Replace with your own connection parameters
var (
	server           = "xtnjypc2pt4f.db-ro.flowlogic.com.au"
	port             = 6033
	user             = "skillsconn_bi"
	password         = "Paothae9Ai0Iezoow9xahNg7"
	dbname           = "flg_skillsconn"
	connectionString = ""
	fromDate         time.Time
	toDate           time.Time
)

const (
	xmlTimeFormat = `2006-01-02T15:04:05`

	LeaveQuery = `SELECT 
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
	HolidaysQuery = `SELECT rs_pubhol_date as Holiday FROM flg_skillsconn.roster_public_holidays`
)

type xmlTime time.Time

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
	Department   string  `csv:"Department"`
}

type LeaveData map[string]map[string]float32

type HolidayData map[time.Time]time.Time

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
	if len(os.Args) < 3 {
		fmt.Println("usage: timeshift <timesheet>.wtc YYYYMMDD [connection_string]")
		fmt.Println("   YYYYMMDD - end of fortnight (fi 20230416: Mon 3th Apr - Sun 16th inclusive)")
		fmt.Println("   that will produce <timesheet>_out.xtc and <timesheet_out>.csv")
		fmt.Println("   connection_string: in format: skillsconn_bi:******@tcp(xtnjypc2pt4f.db-ro.flowlogic.com.au:6033)/flg_skillsconn")
		fmt.Println("       user:password@tcp(server:port)")
		return
	}

	if connectionString == "" && len(os.Args) > 3 {
		connectionString = os.Args[3]
	}

	var err error
	if connectionString == "" && password == "" {
		password, err = credentials()
		if err != nil {
			log.Fatal(fmt.Sprintf("Error getting password: %v", err))
		}
	}

	fromDate, toDate, err = parseDate(os.Args[2])
	if err != nil {
		log.Fatal(fmt.Sprintf("Error parsing interval date: %s %v", os.Args[2], err))
	}

	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	trimFilename := strings.ReplaceAll(fileName, "[", "_")
	trimFilename = strings.ReplaceAll(trimFilename, "]", "_")
	fileNameOut := trimFilename[:len(trimFilename)-len(ext)] + "_out" + ext
	csvFileNameOut := trimFilename[:len(trimFilename)-len(ext)] + "_out" + ".csv"

	leaveData, err := readLeaveData()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading data from DB: %v", err))
	}
	log.Printf("loaded %d staff from DB", len(leaveData))

	HolidayData, err := readHolidays()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading holiday data from DB: %v", err))
	}
	log.Printf("loaded %d holidays from DB", len(HolidayData))

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

	err = processLeaves(timeCards, leaveData, HolidayData)
	if err != nil {
		log.Fatal(fmt.Sprintf("process leaves: %v", err))
	}

	err = addHolidayIfMissing(timeCards, HolidayData, fromDate, toDate, leaveData)
	if err != nil {
		log.Fatal(fmt.Sprintf("add holidays: %v", err))
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

func parseDate(arg string) (time.Time, time.Time, error) {
	toDate, err := time.Parse("20060102", arg)
	fromDate := toDate.AddDate(0, 0, -13)
	return fromDate, toDate, err
}

func sortName(s string) string {
	words := strings.Fields(s)
	sort.Strings(words)
	return strings.Join(words, " ")
}

func getConnection() (*sql.DB, error) {
	rootCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Failed to load system root CA certificates: %v", err)
	}
	err = mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:            rootCertPool,
		InsecureSkipVerify: true,
		// VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		//     fmt.Println("Verifying peer certificate")
		//     return nil
		// },
	})
	if err != nil {
		log.Fatalf("Failed to register custom TLS config: %v", err)
	}
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=custom", user, password, server, port, dbname)
	if connectionString != "" {
		connString = connectionString + "/tls-custom"
	}
	log.Printf("Connection: " + connString)
	return sql.Open("mysql", connString)
}

func readLeaveData() (LeaveData, error) {
	db, err := getConnection()
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
		name = sortName(name)
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

func readHolidays() (HolidayData, error) {
	db, err := getConnection()
	if err != nil {
		log.Fatal("Error creating connection pool: " + err.Error())
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	rows, err := db.Query(HolidaysQuery)
	if err != nil {
		return nil, err
	}
	var dateValue string
	holidayData := make(HolidayData)
	for rows.Next() {
		err := rows.Scan(&dateValue)
		if err != nil {
			return nil, err
		}
		date, err := time.Parse("2006-01-02", dateValue)
		if err != nil {
			return nil, err
		}
		holidayData[date] = date
	}
	return holidayData, nil
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

func (l *Leave) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if strings.EqualFold(l.Type, "Unavailability") ||
		strings.EqualFold(l.Type, "Leave without pay") {
		return nil
	}
	return e.EncodeElement(*l, start)
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
	_, ok := leaveData[sortName(name)]
	return ok
}

func leaveDataHours(leaveData LeaveData,
	name string, fromDate xmlTime, dayOffset int, holidayData HolidayData) (float32, bool, error) {
	date := time.Time(fromDate).Add(time.Hour * time.Duration(24*dayOffset))
	weekDay := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}[date.Weekday()]
	hours, ok := leaveData[sortName(name)][weekDay]
	if holidayData != nil {
		if _, isHoliday := holidayData[date]; isHoliday {
			return hours, true, nil
		}
	}
	if !ok {
		return 0, false, nil
	}
	return hours, false, nil
}

func processLeaves(timeCards *TimeCards, leaveData LeaveData, holidayData HolidayData) error {
	dayOffset := 0
	currentType := ""
	prevFromDate := xmlTime(time.Time{})
	for i, tc := range timeCards.TimeCards {
		isStaff := leaveDataHasStaff(leaveData, tc.EmployeeName)
		for j, leave := range tc.Leave {
			timeCards.TimeCards[i].Leave[j].FromDate = maxDate(leave.FromDate, fromDate)
			timeCards.TimeCards[i].Leave[j].ToDate = minDate(leave.ToDate, toDate)
			leave = timeCards.TimeCards[i].Leave[j]

			if prevFromDate != xmlTime(time.Time{}) {
				if prevFromDate != *leave.FromDate {
					dayOffset = 0
				}
			}
			switch leave.Type {
			case "Time in Lieu":
				if isStaff {
					timeCards.TimeCards[i].Leave[j].Type = "ADO"
				} else {
					timeCards.TimeCards[i].Leave[j].Type = "Accrued Days"
				}
				leave = timeCards.TimeCards[i].Leave[j]
				fallthrough
			case "Annual", "Sick", "Long Service", "Compassionate":
				if isStaff && leave.Hours == 7.6 {
					if currentType != leave.Type {
						dayOffset = 0
					}
					currentType = leave.Type
					hours, isHoliday, err := leaveDataHours(leaveData, tc.EmployeeName, *leave.FromDate, dayOffset, holidayData)
					if err != nil {
						return err
					}
					if isHoliday {
						timeCards.TimeCards[i].Leave[j].Type = "PUBHOL"
					}
					timeCards.TimeCards[i].Leave[j].Hours = hours
				} else if !isStaff {
					// keep unchanged for casuals 'Long Service' and "Accrued Days"
					if leave.Type != "Long Service" && leave.Type != "Accrued Days" {
						timeCards.TimeCards[i].Leave[j].Hours = 0
					}
				}
				dayOffset++
			case "PUBHOL":
				if isStaff && leave.Hours == 7.6 {
					hours, _, err := leaveDataHours(leaveData, tc.EmployeeName, *leave.FromDate, 0, nil)
					if err != nil {
						return err
					}
					timeCards.TimeCards[i].Leave[j].Hours = hours
				}
			default:
				timeCards.TimeCards[i].Leave[j].Hours = 0
			}
			prevFromDate = *leave.FromDate
		}
		dayOffset = 0
		currentType = ""
		prevFromDate = xmlTime(time.Time{})
	}
	return nil
}

func minDate(xmlDate1 *xmlTime, date2 time.Time) *xmlTime {
	date1 := time.Time(*xmlDate1)
	ret := xmlTime(date2)
	if date1.Before(date2) {
		ret = xmlTime(date1)
	}
	return &ret
}
func maxDate(xmlDate1 *xmlTime, date2 time.Time) *xmlTime {
	date1 := time.Time(*xmlDate1)
	ret := xmlTime(date1)
	if date1.Before(date2) {
		ret = xmlTime(date2)
	}
	return &ret
}

func processAllowances(timeCards *TimeCards) ([]CsvShift, error) {
	csvShifts := make([]CsvShift, 0)
	for i, tc := range timeCards.TimeCards {
		allowances := make([]Allowance, 0)
		cntAllowances := 0
		continuesShift := float64(0)
		cntBreaks := 0
		shiftNdx := 0
		carryForward := 0.0
		var previousShift *ShiftHours
		isOvernight := false
		for i, shift := range tc.Shift {
			for j, sh := range shift.ShiftHours {
				isTrainingOrAdmin := sh.Department == "TRAIN" || sh.Department == "92"

				// is new day
				if previousShift != nil && truncateToDay(sh.StartTime) != truncateToDay(previousShift.StartTime) {
					shiftNdx = 0
				}

				if shiftNdx == 0 {
					cntAllowances = 0
					continuesShift = 0
					previousShift = nil
					cntBreaks = 0
					carryForward = 0
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

				if isTrainingOrAdmin {
					carryForward += breakDuration
					breakDuration = 0
				} else {
					breakDuration += carryForward
					carryForward = 0.0
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
					Department:   sh.Department,
				})

				if !isTrainingOrAdmin {
					shiftNdx++
					continuesShift += shiftDuration
				}
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
	if len(csvShifts) == 0 {
		return nil
	}
	b, err := csvutil.Marshal(csvShifts)
	if err != nil {
		fmt.Println("error:", err)
	}
	bb := []byte("{}\n")
	bb = append(bb, b...)
	return ioutil.WriteFile(filename, bb, 0644)
}

// Private function to build a list of holidays that are not on weekends
func getValidHolidays(holidayData HolidayData, fromDate, toDate time.Time) []time.Time {
	var holidays []time.Time
	for date := fromDate; !date.After(toDate); date = date.AddDate(0, 0, 1) {
		if _, isHoliday := holidayData[date]; isHoliday && date.Weekday() != time.Saturday && date.Weekday() != time.Sunday {
			holidays = append(holidays, date)
		}
	}
	return holidays
}

func addHolidayIfMissing(timeCards *TimeCards, holidayData HolidayData, fromDate, toDate time.Time, leaveData LeaveData) error {
	holidays := getValidHolidays(holidayData, fromDate, toDate)
	if len(holidays) == 0 {
		return nil
	}

	for _, date := range holidays {
		for i, tc := range timeCards.TimeCards {
			isStaff := leaveDataHasStaff(leaveData, tc.EmployeeName)
			if isStaff {
				hasShift := false
				hasLeave := false
				shiftHours := 0.0
				for _, shift := range tc.Shift {
					for _, sh := range shift.ShiftHours {
						if truncateToDay(sh.StartTime).Equal(date) {
							hasShift = true
							shiftHours += time.Time(sh.FinishTime).Sub(time.Time(sh.StartTime)).Hours()
						}
					}
				}
				for _, leave := range tc.Leave {
					leaveFromDate := time.Time(*leave.FromDate)
					leaveToDate := time.Time(*leave.ToDate)
					if !leaveFromDate.After(date) && !leaveToDate.Before(date) {
						hasLeave = true
						break
					}
				}
				hours := float32(7.5)
				h, _, err := leaveDataHours(leaveData, tc.EmployeeName, xmlTime(date), 0, nil)
				if err == nil {
					hours = h
				}
				if !hasShift && !hasLeave {
					leave := Leave{
						Type:     "PUBHOL",
						Hours:    hours,
						FromDate: (*xmlTime)(&date),
						ToDate:   (*xmlTime)(&date),
					}
					timeCards.TimeCards[i].Leave = append(timeCards.TimeCards[i].Leave, leave)
				}
				if hasShift {
					msg := fmt.Sprintf("Employee %s logged shift(s) on holiday %s: %f of %f", tc.EmployeeName, date.Format("2006-01-02"), shiftHours, hours)
					if float32(shiftHours) < hours {
						log.Printf("%s -> adjusted %f\n", msg, hours-float32(shiftHours))
						leave := Leave{
							Type:     "PUBHOL",
							Hours:    hours - float32(shiftHours),
							FromDate: (*xmlTime)(&date),
							ToDate:   (*xmlTime)(&date),
						}
						timeCards.TimeCards[i].Leave = append(timeCards.TimeCards[i].Leave, leave)

					} else {
						log.Printf("%s -> no adjustments required\n", msg)
					}
				}
			}
		}
	}
	return nil
}
