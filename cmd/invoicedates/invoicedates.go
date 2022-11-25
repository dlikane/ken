package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jszwec/csvutil"
)

type Invoice struct {
	FirstName       string  `csv:"Client First Name"`
	LastName        string  `csv:"Client Last Name"`
	CardID          string  `csv:"Card ID"`
	Job             string  `csv:"Job"`
	Account         string  `csv:"Account N"`
	TaxCode         string  `csv:"Tax Code"`
	Description     string  `csv:"Description"`
	Date            string  `csv:"Date"`
	Total           string  `csv:"Total"`
	Price           string  `csv:"Price"`
	ItemNumber      string  `csv:"Item Number"`
	Quantity        string  `csv:"Quantity"`
	Category        string  `csv:"Category"`
	CustomerPO      string  `csv:"CustomerPO n."`
	SalesPerson     string  `csv:"Salesperson Name"`
	Refferal        string  `csv:"Referral Source"`
	ServiceDateFrom string  `csv:"Service Date From"`
	ServiceDateTo   string  `csv:"Service Date To"`
	Memo            string  `csv:"Journal Memo"`
	Comments        string  `csv:"Comments"`
	Reference       string  `csv:"Invoice Reference"`
	Tot             float32 `csv:"-"`
	Qty             float32 `csv:"-"`
	MaxQty          float32 `csv:"-"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: invoicedates <invoice>.txt")
		fmt.Println("   that will produce <invoice>_out.txt")
		return
	}

	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	fileNameOut := fileName[:len(fileName)-len(ext)] + "_out" + ext

	invoices, err := readInvoices(fileName)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't parse file: %s: %v", fileName, err))
	}

	err = processInvoices(invoices)
	if err != nil {
		log.Fatal(fmt.Sprintf("process cards: %v", err))
	}

	invoices, err = sumUpInvoices(invoices)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't summup invoices: %v", err))
	}

	err = writeInvoices(invoices, fileNameOut)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't write file: %s: %v", fileNameOut, err))
	}
}

func readInvoices(filename string) ([]Invoice, error) {
	var err error
	var csvInput []byte
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	firstLine := true
	for scanner.Scan() {
		if !firstLine {
			csvInput = append(csvInput, scanner.Text()...)
			csvInput = append(csvInput, "\n"...)
		}
		firstLine = false
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	var invoices []Invoice
	err = csvutil.Unmarshal(csvInput, &invoices)
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

func processInvoices(invoices []Invoice) error {
	cnt := 0
	for i := range invoices {
		inv := invoices[i]
		if inv.Description != "" && inv.ServiceDateFrom != "" && inv.ServiceDateTo != "" {
			if _, ok := codes[inv.ItemNumber]; ok {
				log.Printf("found code")
				from, err := time.Parse("2006-01-02", inv.ServiceDateFrom)
				if err != nil {
					return err
				}
				fromYear, fromMonth, _ := from.Date()
				fromFirstOfMonth := time.Date(fromYear, fromMonth, 1, 0, 0, 0, 0, from.Location())
				invoices[i].ServiceDateFrom = fromFirstOfMonth.Format("2006-01-02")
				to, err := time.Parse("2006-01-02", inv.ServiceDateTo)
				if err != nil {
					return err
				}
				toYear, toMonth, _ := to.Date()
				toFirstOfMonth := time.Date(toYear, toMonth, 1, 0, 0, 0, 0, to.Location())
				toLastOfMonth := toFirstOfMonth.AddDate(0, 1, -1)
				invoices[i].ServiceDateTo = toLastOfMonth.Format("2006-01-02")
			}
			invoices[i].Description += " " + reformatDate(invoices[i].ServiceDateFrom) + " - " + reformatDate(invoices[i].ServiceDateTo)
			qty, err := strconv.ParseFloat(inv.Quantity, 32)
			if err == nil {
				invoices[i].Qty = float32(qty)
			}
			tot, err := strconv.ParseFloat(inv.Total, 32)
			if err == nil {
				invoices[i].Tot = float32(tot)
			}
			cnt++
		}
	}
	log.Printf("number of lines: %d invoices: %d", len(invoices), cnt)
	return nil
}

func sumUpInvoices(invoices []Invoice) ([]Invoice, error) {
	clientInvoices := make(map[string]*Invoice, 1)
	for _, inv := range invoices {
		if inv.CardID == "" {
			continue
		}
		if _, ok := clientInvoices[inv.CardID]; ok {
			clientInvoices[inv.CardID].Qty += inv.Qty
			clientInvoices[inv.CardID].Tot += inv.Tot
			if clientInvoices[inv.CardID].MaxQty < inv.Qty {
				clientInvoices[inv.CardID].MaxQty = inv.Qty
				clientInvoices[inv.CardID].Job = inv.Job
			}
		} else {
			rec := inv
			clientInvoices[inv.CardID] = &rec
			clientInvoices[inv.CardID].MaxQty = rec.Qty
		}
	}

	ret := make([]Invoice, 0)
	for _, inv := range clientInvoices {
		inv.Quantity = fmt.Sprintf("%.2f", inv.Qty)
		inv.Total = fmt.Sprintf("%.2f", inv.Tot)
		ret = append(ret, *inv)
	}
	return ret, nil
}

func writeInvoices(invoices []Invoice, filename string) error {
	b, err := csvutil.Marshal(invoices)
	if err != nil {
		fmt.Println("error:", err)
	}
	bb := []byte("{}\n")
	bb = append(bb, b...)
	return ioutil.WriteFile(filename, bb, 0644)
}

func reformatDate(date string) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		fmt.Printf("error formatting: %s %v\n", date, err)
		return date
	}
	return d.Format("2/01/2006")
}

var codes = map[string]bool{
	"07_002_0106_8_3":    true,
	"01_134_0117_8_1":    true,
	"01_022_0120_1_1":    true,
	"01_020_0120_1_1":    true,
	"01_019_0120_1_1":    true,
	"04_590_0125_6_1":    true,
	"04_591_0136_6_1":    true,
	"04_592_0104_6_1":    true,
	"04_821_0133_6_1":    true,
	"08_590_0106_2_3":    true,
	"09_590_0106_6_3":    true,
	"09_591_0117_6_3":    true,
	"10_590_0102_5_3":    true,
	"10_590_0133_5_3":    true,
	"11_590_0117_7_3":    true,
	"13_590_0102_4_3":    true,
	"NON-NDIS Transport": true,
	"02_051_0108_1_1":    true,
	"Linkt Toll Charge":  true,
	"TAC - Transport":    true,
}
