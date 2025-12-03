#### Using Apache Bench (ab) for Load Testing

ab -n 10000 -c 100 http://localhost:9090/list > analysis.txt

<!-- This sends 100 requests with 10 concurrent users. -->



