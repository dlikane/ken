package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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

type TimeCard struct {
	XMLName      xml.Name `xml:"TimeCard"`
	TimeCardNo   string   `xml:"TimeCardNo"`
	EmployeeName string   `xml:"EmployeeName"`
	Shift        []Shift  `xml:"Shift"`
	Allowance    *Allowance `xml:"Allowance,omitempty"`
	TotalHours   float32  `xml:"TotalHours"`
}

type TimeCards struct {
	XMLName   xml.Name   `xml:"TimeCards"`
	Version   string     `xml:"Version,attr"`
	TimeCards []TimeCard `xml:"TimeCard"`
}

func main() {
	log.Printf("started\n")
	if len(os.Args) < 2 {
		fmt.Println("usage: timeshift <timesheet>.xml")
		fmt.Println("   that will produce <timesheet>_out.xml")
		return
	}

	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	fileNameOut := fileName[:len(fileName)-len(ext)] + "_out" + ext

	timeCards, err := readCards(fileName)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't parse file: %s: %v", fileName, err))
	}

	err = processCards(timeCards)
	if err != nil {
		log.Fatal(fmt.Sprintf("process cards: %v", err))
	}

	err = writeCards(timeCards, fileNameOut)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't write file: %s: %v", fileNameOut, err))
	}
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
	}

	return nil
}
