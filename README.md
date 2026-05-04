# Web Server

Background Services implementation in Go from scratch. The constituent services is - 

## Server
Backend Server that performs the service. 

## API Gateway
The first interface a request hits. It houses the Load Balancer and Rate Limiter

## Load Balancer
Takes in a list of available servers and forwards a request to its suitable server. Algorithms available
 - Round Robin
 - Consistent Hashing

## Rate Limiter
Limit the access to the server based on frequency of request from a pariticular user / ip address. 