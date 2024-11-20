package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Class represents a studio class
type Class struct {
	ID        int    `json:"id"`
	ClassName string `json:"className"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Capacity  int    `json:"capacity"`
}

// Booking represents a booking for a class
type Booking struct {
	ID          int    `json:"id"`
	MemberName  string `json:"memberName"`
	Date        string `json:"date"`
	ClassName   string `json:"className"`
}

var (
	classes    []Class    // Temp Slice to hold class data
	bookings   []Booking  // Temp Slice to hold booking data
	classId    =1         // Incremental ID for classes
	bookingId  =1         // Incremental ID for bookings
	mutex      sync.Mutex // Mutex for thread safety
)

// dataFromJsonFile reads and unmarshals data from a JSON file
func dataFromJsonFile(fileName string, destination interface{}) error {
	// Open or create the file if it doesn't exist
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE,0666)
	if err!=nil {
		return err
	}
	defer file.Close()

	// Read all data from the file
	data, err:= io.ReadAll(file)
	if err != nil {
		return err
	}

	// If the file is empty, skip unmarshalling
	if len(data)==0 {
		return nil      
	}
	return json.Unmarshal(data, destination)
}


// writeDataToJsonFile writes updated data into a JSON file
func writeDataToJsonFile(fileName string, data interface{}) error {
		// Marshal the data into JSON format
	jsonData, err := json.MarshalIndent(data, "", " ")
	if err!= nil {
		return err
	}
	// Write the JSON data into the file
	return os.WriteFile(fileName, jsonData, 0666)
}


// logData writes a log entry for each API call response
func logData(msg string, data interface{}) {
	// Open or create the log file
	logFile, err:= os.OpenFile("api_responses.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Error accessing log file. Error: ",err)
	}
	defer logFile.Close()

	// Prepare the log entry with a timestamp
	logEntry := fmt.Sprintf("[%s] %s: %v\n", time.Now().Format("02-01-2006 15:04:05"), msg, data)
	
	// Write the log entry to the file
	_,err = logFile.WriteString(logEntry)
	if err != nil {
		fmt.Println("Error writing to the Log File, Error: ", err)
	}
}

// successResponse to send a consistent success response
func successResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.WriteHeader(statusCode)

	// a custom response with message and data
	customResponse := struct {
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}{
		Message: message,
		Data:    data,
		
	}

	// Write the response as JSON
	json.NewEncoder(w).Encode(customResponse)
}


// errorResponse to send a consistent error response
func errorResponse(w http.ResponseWriter, statusCode int,message string){
	w.WriteHeader(statusCode)

	// Construct an error response with a message
	response := map[string]interface{}{
		"message" : message,
	}
	// Write the response as JSON
	json.NewEncoder(w).Encode(response)
}


// Handler for class creation
func classHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request method is POST
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	
	// Decode the request body into a Class struct
	var newClass Class
	if err := json.NewDecoder(r.Body).Decode(&newClass); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the class fields
	if newClass.ClassName == "" || newClass.StartDate == "" || newClass.EndDate == "" || newClass.Capacity <= 0 {
		errorResponse(w, http.StatusBadRequest, "Invalid data format")
		return
	}

	// Parse and validate the dates
	startDate, err := time.Parse("02-01-2006", newClass.StartDate)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid startDate format, use DD-MM-YYYY")
		return
	}

	endDate, err := time.Parse("02-01-2006", newClass.EndDate)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid endDate format, use DD-MM-YYYY")
		return
	}

	// Ensure the end date is not before the start date
	if endDate.Before(startDate) {
		errorResponse(w, http.StatusBadRequest, "endDate must be after startDate")
		return
	}

	// Ensure hold on the classes slice temporarily to tackle concurrency
	mutex.Lock()
	defer mutex.Unlock()

	// Assign a unique ID to the class and append it to the classes slice
	newClass.ID = classId
	classId++
	classes = append(classes, newClass)

	// Save classes to JSON file
	if err := writeDataToJsonFile("classes.json", classes); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to save class data")
		return
	}

	// Send a success response and log the event
	successResponse(w, http.StatusCreated, "Class created successfully", newClass)
	logData("Class created successfully", newClass)
}


// Handler for booking a slot in the existing class
func bookingHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure the request method is POST
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	// Decode the request body into a Booking struct
	var newBooking Booking
	if err := json.NewDecoder(r.Body).Decode(&newBooking); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the booking fields
	if newBooking.MemberName == "" || newBooking.Date == "" || newBooking.ClassName == "" {
		errorResponse(w, http.StatusBadRequest, "Invalid field format")
		return
	}

	// Validate the booking fields
	bookingDate, err := time.Parse("02-01-2006", newBooking.Date)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid date format, use DD-MM-YYYY")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// Find the class by name and ensure the date is within its range
	var classFound *Class
	for _, class := range classes {
		if class.ClassName == newBooking.ClassName {
			startDate, _ := time.Parse("02-01-2006", class.StartDate)
			endDate, _ := time.Parse("02-01-2006", class.EndDate)
			if !bookingDate.Before(startDate) && !bookingDate.After(endDate) {
				classFound = &class
				break
			}
		}
	}

	if classFound == nil {
		errorResponse(w, http.StatusBadRequest, "Class is not available on the specified date")
		return
	}

	// Count current bookings for the class on the specified date
	currentBookings := 0
	for _, booking := range bookings {
		if booking.ClassName == newBooking.ClassName && booking.Date == newBooking.Date {
			currentBookings++
		}
	}

	// Calculate available slots and ensure there's availability	
	availableSlots := classFound.Capacity - currentBookings
	if availableSlots <= 0 {
		errorResponse(w, http.StatusBadRequest, "No available slots for the selected class on this date")
		return
	}
	// Assign a unique ID to the booking and append it to the bookings slice
	newBooking.ID = bookingId
	bookingId++
	bookings = append(bookings, newBooking)

	// Save bookings to the JSON file
	if err := writeDataToJsonFile("bookings.json", bookings); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to save booking data")
		return
	}

	// Prepare the response with booking details and available slots
	response := map[string]interface{}{
		"booking":        newBooking,
		"availableSlots": availableSlots - 1,
	}

	// Send a success response and log the event
	successResponse(w, http.StatusCreated, "Booking successful", response)
	logData("Booking successful", response)
}


func main() {
		// Load data from JSON files
		if err := dataFromJsonFile("classes.json", &classes); err != nil {
			fmt.Println("Error loading classes:", err)
		}
	
		if err := dataFromJsonFile("bookings.json", &bookings); err != nil {
			fmt.Println("Error loading bookings:", err)
		}
	
		// Register HTTP handlers
		http.HandleFunc("/classes", classHandler)
		http.HandleFunc("/bookings", bookingHandler)
	
		// Start the HTTP server
		fmt.Println("Listening on :8088")
		http.ListenAndServe(":8088", nil)
}
