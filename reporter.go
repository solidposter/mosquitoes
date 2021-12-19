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
	reqError    int64
	sizeTot     int64
	sizeBig     int64
	sizeSmall   int64
	compression int64
	tcpreuse    int64
	dnsStart    int64
	dnsSuccess  int64
	tlsStart    int64
	tlsSuccess  int64
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
				// fmt.Println(time.Now().Format("20060102 15:04:05.999"), event)
				printSession(event)
			}

			_, isRequest := event["isRequest"]
			if isRequest {
				//	fmt.Println("request report", event)
				rSummary.requests += 1

				if event["error"] == 0 { // update this data only on successful requests
					rSummary.reqTotal += event["timeNano"]
					if rSummary.reqFastest == 0 || event["timeNano"] < rSummary.reqFastest {
						rSummary.reqFastest = event["timeNano"]
					}
					if rSummary.reqSlowest == 0 || event["timeNano"] > rSummary.reqSlowest {
						rSummary.reqSlowest = event["timeNano"]
					}

					rSummary.sizeTot += event["contentLength"]
					if rSummary.sizeBig == 0 || event["contentLength"] > rSummary.sizeBig {
						rSummary.sizeBig = event["contentLength"]
					}
					if rSummary.sizeSmall == 0 || event["contentLength"] < rSummary.sizeSmall {
						rSummary.sizeSmall = event["contentLength"]
					}

					statusCode, _ := event["statusCode"]
					_, ok := rSummary.statusCodes[statusCode]
					if ok {
						rSummary.statusCodes[statusCode] += 1
					} else {
						rSummary.statusCodes[statusCode] = 1
					}

				} else {
					rSummary.reqError += event["error"]
				}

				rSummary.compression += event["compression"]
				rSummary.tcpreuse += event["TCPreuse"]
				rSummary.dnsStart += event["DNSstart"]
				rSummary.dnsSuccess += event["DNSsuccess"]
				rSummary.tlsStart += event["TLSstart"]
				rSummary.tlsSuccess += event["TLSsuccess"]
			}
		}
	}
}

func printRequestSummary(rSummary requestSummary) {
	//	fmt.Println("print report", rSummary)
	if rSummary.requests == 0 {
		return
	}
	t := time.Now()
	fmt.Print(t.Format("15:04:05.99 "))

	fmt.Printf("requests:%v", rSummary.requests)
	fmt.Printf(" error:%v", rSummary.reqError)
	fmt.Printf(" TCP-reuse:%v", rSummary.tcpreuse)
	fmt.Printf(" comp:%v", rSummary.compression)
	avgTimeMilli := rSummary.reqTotal / rSummary.requests / 1000 / 1000
	fmt.Printf(" avg:%vms", avgTimeMilli)
	fmt.Printf(" fastest:%vms", rSummary.reqFastest/1000/1000)
	fmt.Printf(" slowest:%vms", rSummary.reqSlowest/1000/1000)

	avgSizeBytes := rSummary.sizeTot / rSummary.requests
	fmt.Printf(" totContent:%v", rSummary.sizeTot)
	if avgSizeBytes == rSummary.sizeBig {
		fmt.Printf(" reqSize:%v", avgSizeBytes)
	} else {
		fmt.Printf(" reqSize(")
		fmt.Printf("avg:%v", avgSizeBytes)
		fmt.Printf(" big:%v", rSummary.sizeBig)
		fmt.Printf(" small:%v", rSummary.sizeSmall)
		fmt.Printf(")")
	}

	fmt.Printf(" codes(")
	for key, value := range rSummary.statusCodes {
		fmt.Printf(" %v:%v", key, value)
	}
	fmt.Printf(")")

	if rSummary.dnsStart != 0 {
		fmt.Printf(" dns(start:%v success:%v)", rSummary.dnsStart, rSummary.dnsSuccess)
	}
	if rSummary.tlsStart != 0 {
		fmt.Printf(" tls(start:%v success:%v)", rSummary.tlsStart, rSummary.tlsSuccess)
	}

	fmt.Println()
}

func printSession(session map[string]int64) {
	t := time.Now()
	fmt.Print(t.Format("15:04:05.99 "))

	fmt.Print("TCPsession")
	fmt.Printf(" lifetime(s)(requested:%v actual:%v)", session["lifetime"], session["timeNano"]/1000/1000/1000)

	if session["numRequests"] != 0 {
		fmt.Printf(" requests(tot:%v", session["numRequests"])
		fmt.Printf(" comp:%v", session["compRequests"])
		fmt.Printf(" avg:%vms", session["reqSum"]/session["numRequests"]/1000/1000)
		fmt.Printf(" fastest:%vms", session["reqFastest"]/1000/1000)
		fmt.Printf(" slowest:%vms", session["reqSlowest"]/1000/1000)
		fmt.Print(")")
	}

	if session["numRequests"] != 0 {
		fmt.Printf(" tls(start:%v success:%v)", session["TLSstart"], session["TLSsuccess"])
		fmt.Printf(" clientClose:%v", session["clientClose"])
	}
	fmt.Printf(" error:%v", session["error"])

	fmt.Println()
}
