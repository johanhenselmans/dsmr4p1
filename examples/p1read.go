// Simple example program that illustrates how to use the dsmr4p1 library.
package main

import (
	"flag"
	"fmt"
//	"github.com/mhe/dsmr4p1"
	"../../dsmr4p1"
	"github.com/tarm/serial"
	"log"
	"os"
	"time"

	"io"
)

var testfile = flag.String("testfile", "", "Testfile to use instead of serial port")
var ratelimit = flag.Int("ratelimit", 0, "When using a testfile as input, rate-limit the release of P1 telegrams to once every n seconds.")
var device = flag.String("device", "/dev/ttyUSB0", "Serial port device to use")
var baudrate = flag.Int("baud", 9600, "Baud rate to use")
var preDSMR4 = flag.Bool("preDSMR4", true, "Meter is from BeforeDSMR4, does not contain checksum")

func main() {
	fmt.Println("p1read")
	flag.Parse()

	var input io.Reader

	var err error
	boolpreDSMR4 := *preDSMR4

	if *testfile == "" {
	   // for now:
	   c := &serial.Config{Name: *device, Baud: *baudrate, Size: 7, Parity: serial.ParityEven}
	   if boolpreDSMR4{
		c = &serial.Config{Name: *device, Baud: *baudrate, Size: 7, Parity: serial.ParityEven}
	   } else {
		c = &serial.Config{Name: *device, Baud: *baudrate, Size: 8, Parity: serial.ParityNone}
	   }
		input, err = serial.OpenPort(c)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		input, err = os.Open(*testfile)
		if err != nil {
			log.Fatal(err)
		}
		if *ratelimit > 0 {
			input = dsmr4p1.RateLimit(input, time.Duration(*ratelimit)*time.Second)
		}
	}
	ch := dsmr4p1.Poll(input, boolpreDSMR4)
	for t := range ch {
		r, err := t.Parse()
		if err != nil {
			fmt.Println("Error in telegram parsing:", err)
			continue
		}
		
		fmt.Println("Received telegram")
		/*
		timestamp := r["0-0:1.0.0"][0]
		ts, err := dsmr4p1.ParseTimestamp(timestamp)
		if err != nil {
			fmt.Println("Error in time parsing:", err)
			continue
		}
		fmt.Println("Timestamp:", ts)
		*/
		fmt.Println("Electricty power delivered:", r["1-0:1.8.0"][0])
		fmt.Println("Electricty power received: ", r["1-0:2.8.0"][0])
		fmt.Println()
	}
	fmt.Println("Done. Exiting.")
}
