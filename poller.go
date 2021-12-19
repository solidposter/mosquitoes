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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"
)

func poller(id int, lifetime int, interval int, url string, reportQ chan<- map[string]int64, disableCompression bool) {
	var rStart, rEnd time.Time        // request start and end times
	var sStart, sEnd time.Time        // session start and end times
	var rTicker, sTicker *time.Ticker // request and session tickers
	request := make(map[string]int64) // request statistics, see func resetRequest for more info
	request["isRequest"] = 1
	session := make(map[string]int64) // session statistics, see func sessionRequest for more info
	session["isSession"] = 1
	session["lifetime"] = int64(lifetime)

	sTicker = time.NewTicker(time.Duration(lifetime) * time.Second)      // session ticker
	rTicker = time.NewTicker(time.Duration(interval) * time.Millisecond) // request ticker

	tr := &http.Transport{
		MaxConnsPerHost:    1,
		DisableCompression: disableCompression,
	}
	client := &http.Client{Transport: tr}

	clientTrace := &httptrace.ClientTrace{
		// GetConn: func(hostPort string) {
		// 	fmt.Println("starting to create conn:", hostPort)
		// },
		DNSStart: func(info httptrace.DNSStartInfo) {
			request["DNSstart"] = 1
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if info.Err == nil {
				request["DNSsuccess"] = 1
			}
		},
		TLSHandshakeStart: func() {
			request["TLSstart"] = 1

		},
		TLSHandshakeDone: func(state tls.ConnectionState, errmsg error) {
			if state.HandshakeComplete {
				request["TLSsuccess"] = 1
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				request["TCPreuse"] = 1
			} else {
				if sStart.Nanosecond() != 0 {
					sEnd = time.Now()
				}
				session["timeNano"] = sEnd.Sub(sStart).Nanoseconds()
				reportQ <- copyReport(session)
				// reset session and statistics
				sStart = time.Now()
				sTicker.Reset(time.Duration(lifetime) * time.Second)
				resetSession(session)
			}
		},
	}

	for {
		select {
		case <-rTicker.C:
			resetRequest(request)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Printf("NewRequest error:", err)
				os.Exit(1)
			}
			clientTraceCtx := httptrace.WithClientTrace(req.Context(), clientTrace)
			req = req.WithContext(clientTraceCtx)

			rStart = time.Now()
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Do error:", err)
				rEnd = time.Now()
				request["error"] = 1
				request["timeNano"] = rEnd.Sub(rStart).Nanoseconds()
				reportQ <- copyReport(request)
				updateSession(request, session)
				break
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			rEnd = time.Now()
			request["timeNano"] = rEnd.Sub(rStart).Nanoseconds()
			request["statusCode"] = int64(resp.StatusCode)
			if resp.Uncompressed {
				request["compression"] = 1
			}
			request["contentLength"] = int64(len(body))

			reportQ <- copyReport(request)
			updateSession(request, session)

		case <-sTicker.C:
			tr.CloseIdleConnections()
			session["clientClose"] = 1
		}
	}
}

func resetRequest(request map[string]int64) {
	// reset the statistics for a new request
	// all variables used in this map are documented here
	// isRequest = 1 to identify a request map
	request["statusCode"] = 0    // request response code
	request["timeNano"] = 0      // request time in nanoseconds
	request["error"] = 0         // set to 1 when request fails
	request["contentLength"] = 0 // response content length in bytes
	request["compression"] = 0   // 0 no compression, 1 compression
	request["TCPreuse"] = 0      // 0 new connections, 1 reused connection
	request["TLSstart"] = 0      // 1 TLS handshake started
	request["TLSsuccess"] = 0    // 1 TLS handshake successful
	request["DNSstart"] = 0      // 1 DNS request attempted
	request["DNSsuccess"] = 0    // 1 DNS request successful
}

func resetSession(session map[string]int64) {
	// reset the statistics for a new session
	// all variables used in this map are documented here
	// isSession = 1 to identify a session map
	// lifetime = session lifetime in seconds, as specified on startup

	// store information from previous session
	_, ok := session["error"]
	if ok {
		session["prevError"] = session["error"]
	}
	_, ok = session["clientClose"]
	if ok {
		session["prevClientClose"] = session["clientClose"]
	}

	// reset values for a new session
	session["reqFastest"] = 0   // fastest request
	session["reqSlowest"] = 0   // slowest request
	session["reqSum"] = 0       // Total request times
	session["numRequests"] = 0  // number of requests in the session
	session["compRequests"] = 0 // number of requests with compression
	session["clientClose"] = 0  // 1 if session is closed by client, else 0
	session["timeNano"] = 0     // session time in nanoseconds
	session["error"] = 0
	session["TLSstart"] = 0   // number of TLS handshakes started
	session["TLSsuccess"] = 0 // number of TLS handshakes successful
}

func updateSession(request, session map[string]int64) {
	session["numRequests"] += 1
	session["error"] += request["error"]

	session["reqSum"] += request["timeNano"]
	if session["reqFastest"] == 0 || request["timeNano"] < session["reqFastest"] {
		session["reqFastest"] = request["timeNano"]
	}

	if session["reqSlowest"] == 0 || request["timeNano"] > session["reqSlowest"] {
		session["reqSlowest"] = request["timeNano"]
	}

	session["compRequests"] += request["compression"]
	session["TLSstart"] += request["TLSstart"]
	session["TLSsuccess"] += request["TLSsuccess"]
}

// Copy the map that we send via channel to the reporter, so we don't send a reference
func copyReport(input map[string]int64) map[string]int64 {
	output := make(map[string]int64)
	for key, value := range input {
		output[key] = value
	}
	return output
}
