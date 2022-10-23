# SpartanRPC - Distributed RPC Framework

## Project Introduction

Spartan RPC is a lightweight Golang-based distributed remote procedure call(RPC) framework that can efficiently connect different services in and across the system to reduce redundant work in the communication routines with extra support in timeout handling, services load balancing and monitoring for edge devices that have limited computing capacity. 

It is developed based on [Golang](https://github.com/golang/go) 's standard library [net/rpc](https://github.com/golang/go/tree/master/src/net/rpc) and includes all the basic functionality that the library provides:
* JSON-based efficient message encoding and response decoding, data serialization and deserialization
* Server-side server initialization and hosting, service and method registration, RPC request receiving and processing, and RPC response sending
* Client-side client initialization and connecting, RPC call procedure asynchrony and concurrency, RPC request sending and RPC response receiving

More importantly, encapsulating from the existing functionalities, advanced features were designed and implemented to better support different needs of an edge device in edge computing or distributed computing environment that uses RPC:
* RPC timeout handling mechanism to ensure the call is regulated by customized timeout value for connection time and processing time  
* HTTP protocol exchange for supporting HTTP-based web service utilization
* Load balancing utility through random selection and Round Robin scheduling algorithm to dynamically regular the overall loads of each server
* RPC registry that supports service registration, service discovery, and heartbeat monitoring to support distributed system computing environment

