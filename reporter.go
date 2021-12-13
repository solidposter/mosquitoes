package main

//
// Copyright (c) 2021 Tony Sarendal <tony@polarcap.org>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
//

import (
	"fmt"
	"time"
)

type requestSummary struct {
	requests    int64
	reqFastest  int64
	reqSlowest  int64
	reqTotal    int64
	compression int64
	tcpreuse    int64
	statusCodes map[int64]int64
}

func reporter(input <-chan map[string]int64, interval int) {

	event := make(map[string]int64)
	rSummary := requestSummary{}
	rSummary.statusCodes = make(map[int64]int64)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			printRequestSummary(rSummary)
			// fmt.Println("print report", rSummary)
			rSummary = requestSummary{}
			rSummary.statusCodes = make(map[int64]int64)

		case event = <-input:
			_, isSession := event["isSession"]
			if isSession {
				if event["timeNano"] == 0 { // ignore initial session report
					break
				}
				fmt.Println("session report", event)
			}

			_, isRequest := event["isRequest"]
			if isRequest {
				rSummary.requests += 1
				rSummary.reqTotal += event["timeNano"]
				if rSummary.reqFastest == 0 || event["timeNano"] < rSummary.reqFastest {
					rSummary.reqFastest = event["timeNano"]
				}
				if rSummary.reqSlowest == 0 || event["timeNano"] > rSummary.reqSlowest {
					rSummary.reqSlowest = event["timeNano"]
				}
				rSummary.compression += event["compression"]
				rSummary.tcpreuse += event["TCPreuse"]
				//	fmt.Println("request report", event)

				// record all the status codes
				statusCode, _ := event["statusCode"]
				_, ok := rSummary.statusCodes[statusCode]
				if ok {
					rSummary.statusCodes[statusCode] += 1
				} else {
					rSummary.statusCodes[statusCode] = 1
				}

			}
		}
	}
}

func printRequestSummary(rSummary requestSummary) {
	//	fmt.Println("print report", rSummary)
	if rSummary.requests == 0 {
		return
	}
	fmt.Printf("requests: %v", rSummary.requests)
	fmt.Printf(" TCP-reuse: %v", rSummary.tcpreuse)
	fmt.Printf(" comp: %v", rSummary.compression)
	avgTimeMilli := rSummary.reqTotal / rSummary.requests / 1000 / 1000
	fmt.Printf(" avg: %vms", avgTimeMilli)
	fmt.Printf(" fastest: %vms", rSummary.reqFastest/1000/1000)
	fmt.Printf(" slowest: %vms", rSummary.reqSlowest/1000/1000)

	fmt.Printf(" codes(")
	for key, value := range rSummary.statusCodes {
		fmt.Printf(" %v: %v", key, value)
	}
	fmt.Printf(" )")

	fmt.Println()
}
