package main

// UNIBANKERATOR
// A simple .csv parser for Unibank .csv files to calculate fees in a given time period
// Click the ↥ (Télécharger format CSV) from the RELEVE DE TRANSACTIONS RECENTES tab for the account and save the .csv file
// Input is drag-and-drop: drag the .csv file onto the .exe
// Most things that are likely to change can be edited in the constants section before main()

// Current as of July 2023

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/harry1453/go-common-file-dialog/cfd"
	"github.com/harry1453/go-common-file-dialog/cfdutil"
)

// Constants for the file headers. Change these if the headers change in the output files
const dateField string = "Date Trx"    // Transaction Date header
const descField string = "Description" // Transaction Description header
const amntField string = "Debit"       // Transaction Value header

// Date format constants
// See "Golang time.Parse date format" if needing to change these
const dateFormat = "02-Jan-06" // Format of the in-file date
const dateEntry = "2006-01-02" // Format for user-entered dates; default is ISO

// Verbose: Do you want it on?
const verbose = false

// Function for which words to check for that indicate fees
// If new words are added, include as many characters as possible to reduce ambiguity
var feeList []string = []string{"commis.", "frais", "taxes", "timbre", "commission"} // Add new words here as needed

var transferList []string = []string{"Achat et vente de devise"} // Add new words here as needed

// A list of phrases to ignore if they would otherwise be counted as fees
// New words added here should be as specific as possible
var ignoreList []string = []string{} // Add new words here as needed

func main() {

	writeHeader()

	// Get args from the os (i.e. Windows drag and drop)
	args := os.Args[1:]
	argct := len(args)

	var file string

	// Check if a file was supplied by drag and drop or open a file prompt
	switch argct {
	case 0:
		file = openFile()
	case 1:
		file = args[0]
	default:
		fmt.Println("This program can only handle one file at a time.")
		end()
	}

	i := -1
	for i != 0 {
		i = process(file)
	}

}

func process(currFile string) int {
	file, err := os.Open(currFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Run the file through the reader
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // i.e. unspecified number of fields in case they change it

	// Read the header row
	header, err := reader.Read()
	if err == io.EOF {
		fmt.Println("File appears to be empty.")
	} else if err != nil {
		panic(err)
	}

	// Get the index of the columns we need from the header
	colDate := getindex(header, dateField)
	colDesc := getindex(header, descField)
	colAmnt := getindex(header, amntField)

	// Read the rest of the file
	data, err := reader.ReadAll()
	if err != nil {
		fmt.Println("File read error. The file does not appear to be a *.csv file.")
		end()
	}

	// Ask user for dates
	date1, date2 := getDates()
	fmt.Println("Processing transactions from", date1.Format("02 Jan 2006"), "to", date2.Format("02 Jan 2006"))

	var runningTotal float64 = 0 // Total of fee transactions found
	currLnNo := 0                // Current line being processed
	for _, currLine := range data[1:] {
		currLnNo++
		switch verbose {
		case true:
			fmt.Printf("\n")
			fmt.Printf("Processing line %s ...", strconv.Itoa(currLnNo))
		default:
			fmt.Printf("\r")
			fmt.Printf("Processing line %s ...", strconv.Itoa(currLnNo))
		}

		currDate, err := time.Parse(dateFormat, currLine[colDate])
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		if currDate.Compare(date1) >= 0 && currDate.Compare(date2) <= 0 {
			currDesc := currLine[colDesc]
			if containsFee(currDesc) {
				currAmnt, err := strconv.ParseFloat(currLine[colAmnt], 64)
				if err != nil {
					fmt.Printf("Cannot process the amount on line %s", strconv.Itoa(currLnNo))
					panic(err)
				}
				switch verbose {
				case true:
					fmt.Print(strconv.FormatFloat(currAmnt, 'f', 2, 64) + "\n")
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

func containsTransfer(desc string) bool {
	for _, value := range feeList {
		if strings.Contains(desc, value) {
			for _, value := range transferList {
				if strings.Contains(desc, value) {
					return false
				}
			}
			return true
		}
	}
	return false
}

// Checks if the current slice contains a string indicating a fee
func containsFee(desc string) bool {
	for _, value := range feeList {
		if strings.Contains(desc, value) {
			for _, value := range ignoreList {
				if strings.Contains(desc, value) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func end() {
	fmt.Println("Press any key to exit")
	fmt.Scanln()
	os.Exit(0)
}

// Parse user-entered times
func getDates() (time.Time, time.Time) {
	var usrEntry string
	var date1, date2 time.Time

	// Ask for beginning date
	fmt.Println("Enter the beginning and ending dates to process using the format yyyy-mm-dd.")
	fmt.Print("Beginning date: ")
	// var usrDate1 string
	fmt.Scanln(&usrEntry)
	date1 = checkDate(usrEntry)
	usrEntry = "" // clear user input

	// Figure out default end dates, then ask.
	mDate := time.Date(date1.Year(), date1.Month()+1, 0, 0, 0, 0, 0, date1.Location()) // Last day of the month; i.e. 00 Feb == 31 Jan, etc.
	var qDate time.Time
	switch {
	case date1.Day() <= 15:
		qDate = time.Date(date1.Year(), date1.Month(), 15, 0, 0, 0, 0, date1.Location())
	case date1.Day() >= 16:
		qDate = mDate
	}
	fmt.Println("Enter the ending date. You can also enter 'q' to calculate to the end of the quinzaine or 'm' to calculate to the end of the month.")

	// Get ending date with special options
	fmt.Print("Ending date: ")
	fmt.Scanln(&usrEntry)
	switch usrEntry {
	case "q":
		date2 = qDate
	case "m":
		date2 = mDate
	default:
		date2 = checkDate(usrEntry)
	}

	return date1, date2
}

// Asks the user to enter a date using the supplied prompt and returns it as a time.Time object
// If there is an entry error, it will reprompt the user to reenter it until a valid date is entered.
func checkDate(date string) time.Time {
	var rtDate time.Time

	i := -1
	for i != 0 {
		var err error
		rtDate, err = time.Parse(dateEntry, date)
		if err != nil {
			fmt.Print("Entered date is invalid, please try again: ")
			fmt.Scanln(&date)
			i = -1
		} else {
			i = 0
		}
	}
	return rtDate
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

// Open a file selection prompt
func openFile() string {
	userprofile, _ := os.UserHomeDir() // This bypasses an error in cfd when calling %userprofile%
	var (
		file string
		err  error
	)

	os.Setenv("winsymlink", "0")

	i := -1
	for i != 0 {
		file, err = cfdutil.ShowOpenFileDialog(cfd.DialogConfig{
			Title: "Choose a File",
			Role:  "ChooseFile",
			FileFilters: []cfd.FileFilter{
				{
					DisplayName: "CSV Files (*.csv)",
					Pattern:     "*.csv",
				},
				{
					DisplayName: "All Files (*.*)",
					Pattern:     "*.*",
				},
			},
			DefaultFolder:           userprofile + `\Downloads\`,
			SelectedFileFilterIndex: 0,
			FileName:                "",
			DefaultExtension:        "csv",
		})
		if err != nil {
			if err == cfd.ErrorCancelled {
				fmt.Print("File selection cancelled. Enter [o] to open a file or enter any other key to exit: ")
				var key string
				fmt.Scanln(&key)
				switch key {
				case "o":
					i = -1
				default:
					os.Exit(0)
				}
			} else {
				fmt.Printf("Unknown error: %s\n", err)
				fmt.Printf("If you are using a non-Windows operating system try draging and dropping the file onto the program instead")
			}
		}

		i = 0
	}

	return file
}
