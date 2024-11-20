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

type Class struct {
	ID        int    `json:"id"`
	ClassName string `json:"className"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Capacity  int    `json:"capacity"`
}



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

// reading and marshaling data from .json file acting as a database
func dataFromJsonFile(fileName string, destination interface{}) error {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE,0666)
	if err!=nil {
		return err
	}
	defer file.Close()

	data, err:= io.ReadAll(file)
	if err != nil {
		return err
	}

	if len(data)==0 {
		return nil      // Empty File, need not marshal
	}
	return json.Unmarshal(data, destination)
}


// writing updated data into the .json file acting as a database
func writeDataToJsonFile(fileName string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", " ")
	if err!= nil {
		return err
	}
	return os.WriteFile(fileName, jsonData, 0666)
}


// For writing the api response into a log file 
func logData(msg string, data interface{}) {
	logFile, err:= os.OpenFile("api_responses.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Error accessing log file. Error: ",err)
	}
	defer logFile.Close()

	logEntry := fmt.Sprintf("[%s] %s: %v\n", time.Now().Format("02-01-2006 15:04:05"), msg, data)
	_,err = logFile.WriteString(logEntry)
	if err != nil {
		fmt.Println("Error writing to the Log File, Error: ", err)
	}
}

// successResponse to send a consistent success response
func successResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.WriteHeader(statusCode)

	customResponse := struct {
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}{
		Message: message,
		Data:    data,
	}
	json.NewEncoder(w).Encode(customResponse)
}


// errorResponse to send a consistent error response
func errorResponse(w http.ResponseWriter, statusCode int,message string){
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"message" : message,
	}
	json.NewEncoder(w).Encode(response)
}


// Handler for class creation
func classHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var newClass Class
	if err := json.NewDecoder(r.Body).Decode(&newClass); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if newClass.ClassName == "" || newClass.StartDate == "" || newClass.EndDate == "" || newClass.Capacity <= 0 {
		errorResponse(w, http.StatusBadRequest, "Invalid data format")
		return
	}

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

	if endDate.Before(startDate) {
		errorResponse(w, http.StatusBadRequest, "endDate must be after startDate")
		return
	}
	// holding the classes slice temporarily to tackle concurrency
	mutex.Lock()
	defer mutex.Unlock()

	newClass.ID = classId
	classId++
	classes = append(classes, newClass)

	// Save classes to .json file
	if err := writeDataToJsonFile("classes.json", classes); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to save class data")
		return
	}

	successResponse(w, http.StatusCreated, "Class created successfully", newClass)
	logData("Class created successfully", newClass)
}


// Handler for booking a slot in the existing class
func bookingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var newBooking Booking
	if err := json.NewDecoder(r.Body).Decode(&newBooking); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if newBooking.MemberName == "" || newBooking.Date == "" || newBooking.ClassName == "" {
		errorResponse(w, http.StatusBadRequest, "Invalid field format")
		return
	}

	bookingDate, err := time.Parse("02-01-2006", newBooking.Date)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid date format, use DD-MM-YYYY")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

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

	currentBookings := 0
	for _, booking := range bookings {
		if booking.ClassName == newBooking.ClassName && booking.Date == newBooking.Date {
			currentBookings++
		}
	}
	
	availableSlots := classFound.Capacity - currentBookings
	if availableSlots <= 0 {
		errorResponse(w, http.StatusBadRequest, "No available slots for the selected class on this date")
		return
	}

	newBooking.ID = bookingId
	bookingId++
	bookings = append(bookings, newBooking)

	// Save bookings to file
	if err := writeDataToJsonFile("bookings.json", bookings); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to save booking data")
		return
	}

	response := map[string]interface{}{
		"booking":        newBooking,
		"availableSlots": availableSlots - 1,
	}

	successResponse(w, http.StatusCreated, "Booking successful", response)
	logData("Booking successful", response)
}


func main() {
		// Load data from files
		if err := dataFromJsonFile("classes.json", &classes); err != nil {
			fmt.Println("Error loading classes:", err)
		}
	
		if err := dataFromJsonFile("bookings.json", &bookings); err != nil {
			fmt.Println("Error loading bookings:", err)
		}
	
		http.HandleFunc("/classes", classHandler)
		http.HandleFunc("/bookings", bookingHandler)
	
		fmt.Println("Listening on :8088")
		http.ListenAndServe(":8088", nil)
}