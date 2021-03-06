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
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	compPtr := flag.Bool("c", false, "disables compression")
	intervalPtr := flag.Int("i", 1000, "client interval between requests in milliseconds")
	lifetimePtr := flag.Int("l", 300, "session lifetime in seconds")
	numclientPtr := flag.Int("n", 1, "number of client sessions")
	slowstartPtr := flag.Bool("s", false, "slow start, stage the sessions over the session life time instead of client interval")
	urlPtr := flag.String("u", "https://localhost/", "URL to fetch")
	flag.Parse()

	if *lifetimePtr < 1 {
		fmt.Println("Invalid lifetime:", *lifetimePtr)
		os.Exit(1)
	}
	if *intervalPtr < 1 {
		fmt.Println("Invalid interval:", *intervalPtr)
		os.Exit(1)
	}

	fmt.Println("client interval(ms):", *intervalPtr)
	fmt.Println("session lifetime(s):", *lifetimePtr)
	fmt.Println("number of clients:", *numclientPtr)
	fmt.Println("slow start of sessions:", *slowstartPtr)
	fmt.Println("URL to fetch:", *urlPtr)
	fmt.Println("disable compression:", *compPtr)

	reportQ := make(chan map[string]int64, 100)
	go reporter(reportQ, 1)

	for i := 0; i < *numclientPtr; i++ {
		go poller(i, *lifetimePtr, *intervalPtr, *urlPtr, reportQ, *compPtr)
		// staging pollers
		time.Sleep(time.Duration(*intervalPtr / *numclientPtr) * time.Millisecond)
		if *slowstartPtr {
			time.Sleep(time.Duration(*lifetimePtr / *numclientPtr) * time.Second)
		}
	}
	// wait forever
	<-(chan int)(nil)
}
