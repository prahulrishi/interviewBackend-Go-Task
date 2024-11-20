# interviewBackend-Go-Task


The task is to implement two APIs, one to create a class(say Pilates, Lunges, etc..) as an instructor at a gym with the dates and capacity details of the class.
The second API is to book a slot in the class as a member of the gym.

### Steps to Run the code:
`
go run main.go
`


The main.go file contains the API handlers and when run, server listens on port :8088


input for the class creation API looks like :
```
curl -X POST http://localhost:8088/classes \
-H "Content-Type: application/json" \
-d '{
    "className": "Pilates",
    "startDate": "01-12-2024",
    "endDate": "20-12-2024",
    "capacity": 10
  }'
```


and here the expected response is :
```
{
    "message": "Class created successfully",
    "data": {
        "id": 1,
        "className": "Pilates",
        "startDate": "01-12-2024",
        "endDate": "20-12-2024",
        "capacity": 10
    }
}
```



Similarly, sample input for booking API is :

```
curl -X POST http://localhost:8088/bookings \
-H "Content-Type: application/json" \
-d '{
    "memberName": "Rahul R P",
    "date": "16-12-2024",
    "className": "Pilates"
}'
```

and the expected response would be :
```
{
    "message": "Booking successful",
    "data": {
        "availableSlots": 9,
        "booking": {
            "id": 1,
            "memberName": "Rahul R P",
            "date": "16-12-2024",
            "className": "Pilates"
        }
    }
}
```

Unit test cases are included as well.

To run the tests, run the command
`
go test -v
`

and the test response is :

![](https://drive.google.com/file/d/1Z51MlclpvvP7CFLa5Qc1r_HqSSbI8wlN/view?usp=drive_link)


I have used two files, namely. "classes.json" and "bookings.json" to act as a database to log all the class data and the booking data. 



I have maintained an "api_responses.log" file to log all the apicall responses to later verify.
