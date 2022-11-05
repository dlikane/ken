package main

import (
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
	LastName         string `csv:"Co./Last Name"`
	FirstName        string `csv:"First Name"`
	Addr1            string `csv:"Addr 1 - Line 1"`
	Addr2            string `csv:"Addr 1 - Line 2"`
	Addr3            string `csv:"Addr 1 - Line 3"`
	Addr4            string `csv:"Addr 1 - Line 4"`
	Inclusive        string `csv:"Inclusive"`
	Invoice          string `csv:"Invoice No."`
	Date             string `csv:"Date"`
	CustomerPO       string `csv:"Customer PO"`
	Via              string `csv:"Ship Via"`
	Status           string `csv:"Delivery Status"`
	Item             string `csv:"Item Number"`
	Qty              string `csv:"Quantity"`
	Description      string `csv:"Description"`
	Price            string `csv:"Price"`
	Discount         string `csv:"Discount"`
	Total            string `csv:"Total"`
	IncTotal         string `csv:"Inc-Tax Total"`
	Job              string `csv:"Job"`
	Comment          string `csv:"Comment"`
	Memo             string `csv:"Journal Memo"`
	SlLastName       string `csv:"Salesperson Last Name"`
	SlFirstName      string `csv:"Salesperson First Name"`
	ShipDate         string `csv:"Shipping Date"`
	Ref              string `csv:"Referral Source"`
	TaxCode          string `csv:"Tax Code"`
	TaxAmount        string `csv:"Tax Amount"`
	FrAmount         string `csv:"Freight Amount"`
	FrTaxCode        string `csv:"Freight Tax Code"`
	FrTaxAmount      string `csv:"Freight Tax Amount"`
	SlStatus         string `csv:"Sale Status"`
	TermDue          string `csv:"Terms - Payment is Due"`
	TermDiscountDays string `csv:"Terms - Discount Days"`
	TermBalanceDays  string `csv:"Terms - Balance Due Days"`
	TermDiscount     string `csv:"Terms - % Discount"`
	TermCharge       string `csv:"Terms - % Monthly Charge"`
	AmtPaid          string `csv:"Amount Paid"`
	PaymentMethond   string `csv:"Payment Method"`
	PaymentNotes     string `csv:"Payment Notes"`
	NameOnCard       string `csv:"Name On Card"`
	CardNumber       string `csv:"Card Number"`
	AuthCode         string `csv:"Authorisation Code"`
	Bsb              string `csv:"BSB"`
	ActNumber        string `csv:"Account Number"`
	ActName          string `csv:"Drawer/Account Name"`
	ChqNumber        string `csv:"Cheque Number"`
	Category         string `csv:"Category"`
	CardID           string `csv:"Card ID"`
	RecordID         string `csv:"Record ID"`
	DetailDate       string `csv:"Detail Date"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: invoicekv <invoice>.csv")
		fmt.Println("   that will produce <invoice>_out.csv")
		return
	}
	fileName := os.Args[1]
	ext := filepath.Ext(fileName)
	fileNameOut := fileName[:len(fileName)-len(ext)] + "_out" + ext
	invoices, err := readInvoices(fileName)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't parse file: %s: %v", fileName, err))
	}
	outInvoices, err := processInvoices(invoices)

	if err != nil {
		log.Fatal(fmt.Sprintf("process cards: %v", err))
	}
	err = writeInvoices(outInvoices, fileNameOut)
	if err != nil {
		log.Fatal(fmt.Sprintf("Can't write file: %s: %v", fileNameOut, err))
	}
}

func readInvoices(filename string) ([]Invoice, error) {
	buff, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var invoices []Invoice
	err = csvutil.Unmarshal(buff, &invoices)
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

func processInvoices(invoices []Invoice) ([]Invoice, error) {
	cnt := 0
	outmap := make(map[string][]Invoice)
	for i := range invoices {
		cnt++
		inv := invoices[i]
		if inv.CardID == "" {
			continue
		}
		if _, ok := outmap[inv.CardID]; !ok {
			outmap[inv.CardID] = make([]Invoice, 0)
		}
		outmap[inv.CardID] = append(outmap[inv.CardID], inv)
	}
	log.Printf("number of lines: %d invoices: %d", cnt, len(outmap))
	outlist := make([]Invoice, 0)
	toBreak := false
	for _, arr := range outmap {
		if toBreak {
			outlist = append(outlist, Invoice{})
		}
		outlist = append(outlist, mergeInvoices(arr))
		toBreak = true
	}
	return outlist, nil
}

func mergeInvoices(arr []Invoice) Invoice {
	outInvoice := arr[0]
	if len(arr) == 1 {
		return outInvoice
	}
	minDate := arr[0].DetailDate
	maxDate := arr[0].DetailDate
	totalQty := 0.0
	price := 0.0
	for _, inv := range arr {
		if before(inv.DetailDate, minDate) {
			minDate = inv.DetailDate
		}
		if before(maxDate, inv.DetailDate) {
			maxDate = inv.DetailDate
		}
		qty, err := strconv.ParseFloat(inv.Qty, 64)
		if err != nil {
			log.Fatalf("Can't parse QTY: %s for invoice: %v", inv.Qty, inv)
		}
		totalQty += qty
		invPrice, err := strconv.ParseFloat(inv.Price, 64)
		if err != nil {
			log.Fatalf("Can't parse PRICE: %s for invoice: %v", inv.Price, inv)
		}
		if price == 0.0 {
			price = invPrice
		} else if price != invPrice {
			log.Fatal("Expect the same price: %.2f <> %.2f", price, invPrice)
		}
	}
	outInvoice.Qty = floatToString(totalQty)
	outInvoice.Total = fmt.Sprintf("%.2f", totalQty * price)
	outInvoice.Description = outInvoice.Description + " " + minDate + " - " + maxDate
	outInvoice.DetailDate = minDate + " - " + maxDate
	return outInvoice
}

func before(d1 string, d2 string) bool {
	dd1, err := time.Parse("2/01/2006", d1)
	if err != nil {
		log.Fatalf("Can't parse date: %s", d1)
	}
	dd2, err := time.Parse("2/01/2006", d2)
	if err != nil {
		log.Fatalf("Can't parse date: %s", d2)
	}
	return dd1.Before(dd2)
}

func writeInvoices(invoices []Invoice, filename string) error {
	b, err := csvutil.Marshal(invoices)
	if err != nil {
		fmt.Println("error:", err)
	}
	return ioutil.WriteFile(filename, b, 0644)
}

func floatToString(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
}