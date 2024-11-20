package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

// resetTestFiles ensures empty JSON Files
func resetTestFiles() {
	// Replace with temporary file abstraction or mock logic for cleaner testing.
	os.WriteFile("classes.json", []byte("[]"), 0666)
	os.WriteFile("bookings.json", []byte("[]"), 0666)
}

// setupTestEnvironment initializes the test environment by resetting data
func setupTestEnvironment() {
	// Resets test files and in-memory data structures to avoid cross-test contamination.
	resetTestFiles()
	classes = []Class{}
	bookings = []Booking{}
	classId = 1
	bookingId = 1
	mutex = sync.Mutex{}
}
// TestClassHandler verifies the behavior of the class creation handler.
func TestClassHandler(t *testing.T) {
	setupTestEnvironment() // Ensure a clean state before running tests

	// Define multiple test cases for the `/classes` endpoint.
	tests := []struct {
		name       string
		input      Class
		statusCode int
		message    string
	}{
		{
			name: "Valid Class Creation",
			input: Class{
				ClassName: "Yoga",
				StartDate: "01-12-2024",
				EndDate:   "31-12-2024",
				Capacity:  20,
			},
			statusCode: http.StatusCreated,
			message:    "Class created successfully",
		},
		{
			name: "Invalid Dates",
			input: Class{
				ClassName: "Pilates",
				StartDate: "31-12-2024",
				EndDate:   "01-12-2024",
				Capacity:  10,
			},
			statusCode: http.StatusBadRequest,
			message:    "endDate must be after startDate",
		},
		{
			name: "Negative Capacity",
			input: Class{
				ClassName: "Dance",
				StartDate: "10-12-2024",
				EndDate:   "20-12-2024",
				Capacity:  -5,
			},
			statusCode: http.StatusBadRequest,
			message:    "Invalid data format",
		},
	}

	// Iterate through each test case.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert the test input into a JSON body.
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/classes", bytes.NewReader(body))
			rec := httptest.NewRecorder()  // Capture the handler's response.

			classHandler(rec, req)

			// Verify the HTTP status code matches the expected value.
			if rec.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, rec.Code)
			}

			var response map[string]interface{}
			json.NewDecoder(rec.Body).Decode(&response)

			if response["message"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, response["message"])
			}
		})
	}
}
// TestBookingHandler validates the behavior of the bookinga handler.
func TestBookingHandler(t *testing.T) {
	setupTestEnvironment()

	// Pre-create a class to allow bookings against it.
	classes = append(classes, Class{
		ID:        1,
		ClassName: "Pilates",
		StartDate: "15-12-2024",
		EndDate:   "20-12-2024",
		Capacity:  10,
	})
	// Save the pre-created class to the JSON file.
	writeDataToJsonFile("classes.json", classes)

	// Define multiple test cases for the `/bookings` endpoint.
	tests := []struct {
		name       string
		input      Booking
		statusCode int
		message    string
	}{
		{
			name: "Valid Booking",
			input: Booking{
				MemberName: "John Doe",
				Date:       "16-12-2024",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusCreated,
			message:    "Booking successful",
		},
		{
			name: "Class Not Available",
			input: Booking{
				MemberName: "Jane Doe",
				Date:       "25-12-2024",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusBadRequest,
			message:    "Class is not available on the specified date",
		},
		{
			name: "Invalid Date Format",
			input: Booking{
				MemberName: "Alice",
				Date:       "12/16/2024",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusBadRequest,
			message:    "Invalid date format, use DD-MM-YYYY",
		},
		{
			name: "No Slots Available",
			input: Booking{
				MemberName: "Exceeding Slots",
				Date:       "16-12-2024",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusBadRequest,
			message:    "No available slots for the selected class on this date",
		},
	}

	// Iterate through each test case.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill the slots if testing "No Slots Available"
			if tt.name == "No Slots Available" {
				for i := 0; i < 10; i++ {
					bookings = append(bookings, Booking{
						ID:         bookingId,
						MemberName: fmt.Sprintf("Member %d", i),
						Date:       "16-12-2024",
						ClassName:  "Pilates",
					})
					bookingId++
				}
				// Save the filled bookings to the JSON file.
				writeDataToJsonFile("bookings.json", bookings)
			}

			// Convert the booking input into a JSON body.
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(body))
			rec := httptest.NewRecorder()  // Capture the handler's response.

			bookingHandler(rec, req)


			// Verify the HTTP status code matches the expected value.
			if rec.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, rec.Code)
			}

			// Decode the response to validate the message.
			var response map[string]interface{}
			json.NewDecoder(rec.Body).Decode(&response)

			if response["message"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, response["message"])
			}
		})
	}
}

// TestAdditionalEdgeCases verifies the behaviour of booking handler during edge case scenarios
func TestAdditionalEdgeCases(t *testing.T) {
	setupTestEnvironment()

	// Define multiple test cases for the `/bookings` endpoint.
	tests := []struct {
		name       string
		input      Booking
		statusCode int
		message    string
	}{
		{
			name: "Booking with Empty MemberName",
			input: Booking{
				MemberName: "",
				Date:       "16-12-2024",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusBadRequest,
			message:    "Invalid field format",
		},
		{
			name: "Booking with Past Date",
			input: Booking{
				MemberName: "John",
				Date:       "10-10-2020",
				ClassName:  "Pilates",
			},
			statusCode: http.StatusBadRequest,
			message:    "Class is not available on the specified date",
		},
		{
			name: "Overlapping Classes with Different Names",
			input: Booking{
				MemberName: "John",
				Date:       "16-12-2024",
				ClassName:  "Dance",
			},
			statusCode: http.StatusBadRequest,
			message:    "Class is not available on the specified date",
		},
	}

	// Iterate through each test case.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert the test input into a JSON body.
			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(body))
			rec := httptest.NewRecorder()  // Capture the handler's response.

			// Call the booking handler.
			bookingHandler(rec, req)

			// Verify the HTTP status code matches the expected value.
			if rec.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, rec.Code)
			}

			// Decode the response to validate the message.
			var response map[string]interface{}
			json.NewDecoder(rec.Body).Decode(&response)

			if response["message"] != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, response["message"])
			}
		})
	}
}
