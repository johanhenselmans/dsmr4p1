// Simple example program that illustrates how to use the dsmr4p1 library.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	//import the Paho Go MQTT library
	MQTT "github.com/eclipse/paho.mqtt.golang"
	//	"github.com/mhe/dsmr4p1"
	"../../dsmr4p1"
	"github.com/tarm/serial"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var testfile = flag.String("testfile", "", "Testfile to use instead of serial port")
var ratelimit = flag.Int("ratelimit", 0, "When using a testfile as input, rate-limit the release of P1 telegrams to once every n seconds.")
var device = flag.String("device", "/dev/ttyUSB0", "Serial port device to use")
var baudrate = flag.Int("baud", 9600, "Baud rate to use")
var preDSMR4 = flag.Bool("preDSMR4", true, "Meter is from BeforeDSMR4, does not contain checksum")

type WoodyZappRequestMessage struct {
	Version       string
	HardwareId    string
	Timestamp_UTC string // Amount of milliseconds to pause on each send to give TinyG time to send us a qr report
	RequestId     string
	Action        string
	Message       string
	ResponseTopic string
}

func main() {
	fmt.Println("p1read")
	flag.Parse()

	var input io.Reader
	var err error

	boolpreDSMR4 := *preDSMR4

	if *testfile == "" {
		// for now:
		c := &serial.Config{Name: *device, Baud: *baudrate, Size: 7, Parity: serial.ParityEven}
		if boolpreDSMR4 {
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

	opts := MQTT.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetClientID("woodyzapp-dsmrreader")

	//create and start a client using the above ClientOptions
	mqttclient := MQTT.NewClient(opts)
	if token := mqttclient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	boolpreDSMR4 = *preDSMR4

	// See https://github.com/goiot/devices/wiki/Cleanly-exiting-a-program  for cleanly exiting program
	// channel to push to if we want to exit in a clean way
	quitCh := make(chan bool)

	// catch signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	// monitor for signals in the background
	go func() {
		s := <-sigCh
		fmt.Println("\nreceived signal:", s)
		quitCh <- true
	}()

	ch := dsmr4p1.Poll(input, boolpreDSMR4)
	//	ch := dsmr4p1.Poll(input)

	for {
		select {
		case <-quitCh:
			fmt.Println("Done. Exiting.")

		case <-ch:
			for t := range ch {
				tmpdate:= time.Now().UTC()
				r, err := t.Parse()
				if err != nil {
					fmt.Println(tmpdate," Error in telegram parsing:", err)
					continue
				}
				fmt.Println(tmpdate, " Received telegram")
				/*
					timestamp := r["0-0:1.0.0"][0]
					ts, err := dsmr4p1.ParseTimestamp(timestamp)
					if err != nil {
						fmt.Println("Error in time parsing:", err)
						continue
					}
					fmt.Println("Timestamp:", ts)
				*/
				fmt.Println(tmpdate," Electricty power delivered:", r["1-0:1.8.0"][0])
				fmt.Println(tmpdate, "Electricty power received: ", r["1-0:2.8.0"][0])

				type PowerUsageMessage struct {
					PowerDelivered string
					PowerReceived  string
					SensorType     string
				}
				woodytempMessage := PowerUsageMessage{}
				woodytempMessage.PowerDelivered = fmt.Sprintf("%s", r["1-0:1.8.0"][0])
				woodytempMessage.PowerReceived = fmt.Sprintf("%s", r["1-0:2.8.0"][0])
				woodytempMessage.SensorType = fmt.Sprintf("%s", "ISKRA DSMRPre4.0")
				woodyusagejson, _ := json.Marshal(woodytempMessage)
				woodyMessage := WoodyZappRequestMessage{Version: "v1", Timestamp_UTC: strconv.FormatInt(time.Now().UTC().UnixNano(), 10), Action: "/status/PowerUsage", Message: string(woodyusagejson), ResponseTopic: ""}
				text, _ := json.Marshal(woodyMessage)
				token := mqttclient.Publish("/woodyzapp/hal/status/PowerUsage", 0, false, text)
				token.Wait()

			}

		}
	}
}
