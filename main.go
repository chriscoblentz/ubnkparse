package main

// UNIBANKERATOR
// A simple .csv parser for Unibank .csv files to calculate fees in a given time period
// Click the ↥ (Télécharger format CSV) from the RELEVE DE TRANSACTIONS RECENTES tab for the account and save the .csv file
// Input is drag-and-drop: drag the .csv file onto the .exe
// Most things that are likely to change can be edited in the constants section before main()

//Current as of July 2023

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

	for _, currFile := range args {
		// fileCount = fileCount + 1
		// fileCountStr := strconv.Itoa(fileCount)
		// fmt.Println("Processing " + filepath.Base(currFile) + " (" + fileCountStr + " of " + numFilesStr + ")…")
		i := -1
		for i != 0 {
			i = process(currFile)
		}
	}
}

func process(currFile string) int {
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
	fmt.Println()

	fmt.Print("Enter [c] to continue with new dates or enter any other key to exit: ")
	var key string
	fmt.Scanln(&key)
	switch key {
	case "c":
		fmt.Println("=============================")
		fmt.Println()
		return -1
	default:
		return 0
	}
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

func end() {
	fmt.Println("Press any key to exit")
	fmt.Scanln()
}

// Parse user-entered times
func getDates() (time.Time, time.Time) {

	//Ask for beginning date
	fmt.Println("Enter the beginning and ending dates to process using the format yyyy-mm-dd.")
	date1 := checkDate("Beginning Date: ")

	//Figure out default end dates, then ask.
	mDate := time.Date(date1.Year(), date1.Month()+1, 0, 0, 0, 0, 0, date1.Location()) //Last day of the month; i.e. 00 Feb == 31 Jan, etc.
	var qDate time.Time
	switch {
	case date1.Day() <= 15:
		qDate = time.Date(date1.Year(), date1.Month(), 15, 0, 0, 0, 0, date1.Location())
	case date1.Day() >= 16:
		qDate = mDate
	}
	fmt.Println("Enter the ending date. You can also enter 'q' to calculate to the end of the quinzaine or 'm' to calculate to the end of the month.")

	//Was supposed to use checkDate, but
	i := -1
	var usrDate string
	var date2 time.Time
	for i != 0 {
		fmt.Print("Ending date: ")
		fmt.Scanln(&usrDate)
		switch usrDate {
		case "q":
			date2 = qDate
			i = 0
		case "m":
			date2 = mDate
			i = 0
		default:
			rtDate, err := time.Parse(dateEntry, usrDate)
			switch err != nil {
			case true:
				fmt.Println("Entered date is invalid, please try again.")
				i = -1
			case false:
				date2 = rtDate
				i = 0
			}
		}
	}

	return date1, date2
}

// Asks the user to enter a date using the supplied prompt and returns it as a time.Time object
// If there is an entry error, it will reprompt the user to reenter it until a valid date is entered.
func checkDate(prompt string) time.Time {
	var usrDate string
	i := -1
	for i != 0 {
		fmt.Print(prompt)
		fmt.Scanln(&usrDate)
		rtDate, err := time.Parse(dateEntry, usrDate)
		switch err != nil {
		case true:
			fmt.Println("Entered date is invalid, please try again.")
			i = -1
		case false:
			return rtDate
		}
	}
	return time.Now() //Do not understand why we need a return here since it will loop until it gets a correct date in the switch
}

func writeHeader() {
	fmt.Printf("\n")
	fmt.Println("  ██╗ ██╗ ██╗   ██╗███╗   ██╗██╗██████╗  █████╗ ███╗   ██╗██╗  ██╗███████╗██████╗  █████╗ ████████╗ ██████╗ ██████╗ ")
	fmt.Println(" ████████╗██║   ██║████╗  ██║██║██╔══██╗██╔══██╗████╗  ██║██║ ██╔╝██╔════╝██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗")
	fmt.Println(" ╚██╔═██╔╝██║   ██║██╔██╗ ██║██║██████╔╝███████║██╔██╗ ██║█████╔╝ █████╗  ██████╔╝███████║   ██║   ██║   ██║██████╔╝")
	fmt.Println(" ████████╗██║   ██║██║╚██╗██║██║██╔══██╗██╔══██║██║╚██╗██║██╔═██╗ ██╔══╝  ██╔══██╗██╔══██║   ██║   ██║   ██║██╔══██╗")
	fmt.Println(" ╚██╔═██╔╝╚██████╔╝██║ ╚████║██║██████╔╝██║  ██║██║ ╚████║██║  ██╗███████╗██║  ██║██║  ██║   ██║   ╚██████╔╝██║  ██║")
	fmt.Println("  ╚═╝ ╚═╝  ╚═════╝ ╚═╝  ╚═══╝╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝")
	fmt.Printf("\n")
}
