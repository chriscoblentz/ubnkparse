package main

// UNIBANKERATOR
// A simple .csv parser for Unibank .csv files to calculate fees in a given time period
// Click the ↥ (Télécharger format CSV) from the RELEVE DE TRANSACTIONS RECENTES tab for the account and save the .csv file
// Input is drag-and-drop: drag the .csv file onto the .exe
// Most things that are likely to change can be edited in the constants section before main()

//Current as of 01 July 2023

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Constants for the file headers. Change these if the headers change in the output files
const dateField string = "Date Trx"    //Transaction Date header
const descField string = "Description" //Transaction Description header
const amntField string = "Debit"       //Transaction Value header

// Date format constants
// See "Golang time.Parse date format" if needing to change these
const dateFormat = "02-Jan-06" //Format of the in-file date
const dateEntry = "2006-01-02" //Format for user-entered dates; default is ISO

// Verbose: Do you want it on?
const verbose = false

// Function for which words to check for that indicate fees
// If new words are added, include as many characters as possible to reduce ambiguity
var feeList []string = initFeeList()

func initFeeList() []string {
	return []string{"commis.", "frais", "taxes", "timbre", "commissions"} //Add new words here as needed
}

func main() {

	writeHeader()

	//Get args from the os (i.e. Windows drag and drop)
	args := os.Args[1:]
	argct := len(args)

	//Check number of args received to make sure we received exactly one file.
	//Ideally no args would open a file open ui, but there's nothing in the standard library and we're trying to avoid going outside that
	//Could add option to process multiple files, but would probably be confusing anyway
	switch {
	case argct < 1:
		fmt.Println("This program is designed for drag-and-drop. Please drag the .csv file onto the program.")
		end()
	case argct > 1:
		fmt.Println("This program can only handle one file at a time.")
		end()
	}

	// fileCount := 0
	for _, currFile := range args {
		// fileCount = fileCount + 1
		// fileCountStr := strconv.Itoa(fileCount)
		// fmt.Println("Processing " + filepath.Base(currFile) + " (" + fileCountStr + " of " + numFilesStr + ")…")
		process(currFile)
	}
}

func process(currFile string) {
	file, err := os.Open(currFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	//Run the file through the reader
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 //i.e. unspecified number of fields in case they change it

	//Read the header row
	header, err := reader.Read()
	if err == io.EOF {
		log.Println("File appears to be empty.")
		//Break?
	} else if err != nil {
		panic(err)
	}

	//Get the index of the columns we need from the header
	colDate := getindex(header, dateField)
	colDesc := getindex(header, descField)
	colAmnt := getindex(header, amntField)

	//Read the rest of the file
	data, err := reader.ReadAll()
	if err != nil {
		fmt.Println("File read error. The file does not appear to be a *.csv file.")
		end()
	}

	//Ask user for dates
	date1, date2 := getDates()
	fmt.Println("Processing transactions from", date1.Format("02 Jan 2006"), "to", date2.Format("02 Jan 2006"))

	var runningTotal float64 = 0 //Total of fee transactions found
	currLnNo := 0                //Current line being processed
	for _, currLine := range data[1:] {
		currLnNo += 1
		switch verbose {
		case true:
			fmt.Printf("\n")
			fmt.Print("Processing line " + strconv.Itoa(currLnNo) + "… ")
		default:
			fmt.Printf("\r")
			fmt.Printf("Processing line " + strconv.Itoa(currLnNo) + "…")
		}

		currDate, err := time.Parse(dateFormat, currLine[colDate])
		if err != nil {
			log.Println(err)
			panic(err)
		}

		if currDate.Compare(date1) >= 0 && currDate.Compare(date2) <= 0 {
			currDesc := currLine[colDesc]
			if containsFee(currDesc) {
				currAmnt, err := strconv.ParseFloat(currLine[colAmnt], 64)
				if err != nil {
					log.Println("Cannot process the amount on line", currLnNo)
					panic(err)
				}
				switch verbose {
				case true:
					fmt.Print(strconv.FormatFloat(currAmnt, 'f', 2, 64))
				}
				runningTotal += currAmnt
			}

		}
	}
	switch verbose {
	case true:
		fmt.Printf("\n")
	case false:
		fmt.Printf("\r")
	}
	fmt.Println("Processed ", currLnNo, "lines")
	fmt.Println("=============================")
	fmt.Println("TOTAL:", strconv.FormatFloat(runningTotal, 'f', 2, 64))
	fmt.Println("\nPress the Enter Key to end")
	fmt.Scanln()
}

// Gets the index for a string (i.e. for the header row)
func getindex(row []string, seek string) int {
	for index, value := range row {
		if value == seek {
			return index
		}
	}
	return -1
}

// Checks if the current slice contains a string inidcating a fee
func containsFee(desc string) bool {
	for _, value := range feeList {
		if strings.Contains(desc, value) {
			return true
		}
	}
	return false
}

func checkdate(string) {

}

func end() {
	fmt.Println("Press ENTER to exit")
	fmt.Scanln()
}

// Parse user-entered times
func getDates() (time.Time, time.Time) {
	var usrDate1, usrDate2 string

	//Ask for beginning date
	fmt.Println("Enter the beginning and ending dates to process using the format yyyy-mm-dd.")
	fmt.Print("Beginning date: ")
	fmt.Scanln(&usrDate1)
	date1, err := time.Parse(dateEntry, usrDate1)

	if err != nil {
		fmt.Println("Date is invalid.")
		end()
	}

	//Figure out default end dates, then ask.
	mDate := time.Date(date1.Year(), date1.Month()+1, 0, 0, 0, 0, 0, date1.Location()) //Last day of the month; i.e. 00 Feb == 31 Jan, etc.
	var qDate2 time.Time
	switch {
	case date1.Day() <= 15:
		qDate2 = time.Date(date1.Year(), date1.Month(), 15, 0, 0, 0, 0, date1.Location())
	case date1.Day() >= 16:
		qDate2 = mDate
	}

	fmt.Println("Enter the ending date. You can also enter 'q' to calculate to the end of the quinzaine or 'm' to calculate to the end of the month.")
	fmt.Print("Ending date: ")
	fmt.Scanln(&usrDate2)
	var date2 time.Time
	switch usrDate2 {
	case "q":
		date2 = qDate2
	case "m":
		date2 = mDate
	default:
		date2, _ = time.Parse(dateEntry, usrDate2)
		if err != nil {
			fmt.Println("Date is invalid.")
			end()
		}
	}

	return date1, date2
}
func writeHeader() {
	fmt.Printf("\n")
	fmt.Println("  ██╗ ██╗ ██╗   ██╗███╗   ██╗██╗██████╗  █████╗ ███╗   ██╗██╗  ██╗███████╗██████╗  █████╗ ████████╗ ██████╗ ██████╗ ")
	fmt.Println(" ████████╗██║   ██║████╗  ██║██║██╔══██╗██╔══██╗████╗  ██║██║ ██╔╝██╔════╝██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗")
	fmt.Println(" ╚██╔═██╔╝██║   ██║██╔██╗ ██║██║██████╔╝███████║██╔██╗ ██║█████╔╝ █████╗  ██████╔╝███████║   ██║   ██║   ██║██████╔╝")
	fmt.Println(" ████████╗██║   ██║██║╚██╗██║██║██╔══██╗██╔══██║██║╚██╗██║██╔═██╗ ██╔══╝  ██╔══██╗██╔══██║   ██║   ██║   ██║██╔══██╗")
	fmt.Println(" ╚██╔═██╔╝╚██████╔╝██║ ╚████║██║██████╔╝██║  ██║██║ ╚████║██║  ██╗███████╗██║  ██║██║  ██║   ██║   ╚██████╔╝██║  ██║")
	fmt.Println("  ╚═╝ ╚═╝  ╚═════╝ ╚═╝  ╚═══╝╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝")
}
