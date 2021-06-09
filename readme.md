### Jaeger setup example

This example contains two services (flights and passengers) with two different
mysql instances. 

#### Running
1. ``git clone https://github.com/Jepzter/jaeger-example.git``
2. Run ``docker-compose up`` in jaeger-example directory
3. Make a api call to fetch the flights and it's passenger with 
```GET http://localhost:8080/api/flight-service/flight-v1/1```
4. Navigate to to [jaeger ui](http://localhost:16686/search)
5. Select ``flight-service`` or ``passenger-service`` in the search to see all spans created

#### Tracing
