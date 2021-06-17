# Simple jaeger and golang example

This example contains two services (flights and passengers) with two different
mysql instances (flights and passengers). Making any interactions with the API of any service
will report it to [jaeger](https://www.jaegertracing.io/). 

Learn more about OpenTelemetry [here](https://opentelemetry.io/)

## Setup
**_Use host port for curl requests from host machine_**

| Name               | Type                   | Host Port                  | Container port         |
| :---               | :---                   | :---                       | :---                   |
| jaeger             | Jaeger:all-in-one      | 6831/udp, 14269, 16686     | 6831/udp, 14269, 16686 |
| flight-db          | MySQL                  | 3306                       | 3306                   |
| passenger-db       | MySQL                  | 3307                       | 3306                   |
| flight-service     | Go service             | 8080                       | 8080                   |
| passenger-service  | Go service             | 8090                       | 8080                   |

## Running
**_Running this project requires [docker](https://www.docker.com/) and [docker-compose](https://docs.docker.com/compose/install/) version 1.29.2._** 

1. ``git clone https://github.com/Jepzter/jaeger-example.git``
2. **Optional** Choose in docker-compose if you want passenger-service to be unstable and simulate errors. 
Set environment UNSTABLE=true. Default is false 
3. Run ``docker-compose up`` in jaeger-example directory
4. Make a request defined in any of the apis below 
5. Navigate to to [jaeger ui](http://localhost:16686/search)
6. Select ``flight-service`` or ``passenger-service`` in the `services` dropdown menu
7. Click `Find traces`
8. Select a trace to view it's timeline

## Flight service API
### Get flight
```http
GET http://localhost:8080/api/flight-service/flight-v1/1
``` 

| Parameter   | Type      | Description |
| :----       | :---      | :---        |
| `Flight ID` | `integer` | **Required**. The ID of the flight. Example contains only ID 1 and 2 |

#### Response
```json
{
    "FlightID": 1,
    "Name": "FLIGHT 33",
    "Destination": "VXO",
    "Passengers": [
        {
            "PassengerID": 1,
            "FlightID": 1,
            "Firstname": "Jesper",
            "Surname": "Placeholdersson"
        }
    ]
}
```

## Passenger service API
### Get passengers
```http
GET http://localhost:8090/api/passenger-service/passenger-v1?flightId=1
``` 

| Parameter | Type | Description |
| :----     | :--- | :---        |
| `FlightId` | `integer` | **Optional**. The ID of the flight. Example contains only ID 1 and 2 |

#### Response
```json
[
    {
        "PassengerID": 1,
        "FlightID": 1,
        "Firstname": "Jesper",
        "Surname": "Placeholdersson"
    }
]
```
