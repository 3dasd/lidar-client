package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tarm/serial"
)

// readSerial reads the serial connection and forwards to
// either the standard output or the result file.
func readSerial(p *serial.Port, fileWriterChan chan<- string) {
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		t := scanner.Text()
		if strings.HasPrefix(t, "p") {
			fileWriterChan <- t
		} else {
			fmt.Printf("> %s\n", t)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// readStdin reads the standard input and forwards to the scanner via
// serial connection.
func readStdin(p *serial.Port, newFileChan chan<- bool) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		t := scanner.Text()
		// 'r' = run command, need to create a new file to keep the results
		if strings.HasPrefix(t, "r") {
			newFileChan <- true
		}
		p.Write([]byte(t))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// fileWriter manages the files that contain the scan resuls.
//
// newFileChan: send on this channel on new measurements to open new files
// fileWriterChan: everything sent on this is written to the current file
//                 (first character cropped)
// stopChan: fileWriter will finish when this is closed
// stoppedChan: fileWriter will close this once it has finished
func fileWriter(newFileChan <-chan bool, fileWriterChan <-chan string,
	stopChan chan struct{}, stoppedChan chan struct{}) {

	var f *os.File

	// Destructor logic, closes current file
	defer func() {
		if f != nil {
			f.Close()
		}
		close(stoppedChan)
	}()

	for {
		select {
		case <-newFileChan:
			if f != nil {
				f.Close()
			}
			fileName := "points-" + time.Now().Format("2006-01-02-15-04-05") + ".asdp"
			fmt.Printf(">> Opening new file for results: %s\n", fileName)
			var err error
			f, err = os.Create(fileName)
			if err != nil {
				log.Fatalf("unable to open output file: %v", err)
			}
		case s := <-fileWriterChan:
			if f == nil {
				log.Fatalf("Trying to write points file before opening it! Scanner sending wrong messages? Message was: %s", s)
			}
			f.WriteString(s[1:])
			f.WriteString("\n")
		case <-stopChan:
			return
		}
	}
}

var serialPort = flag.String("serial", "/dev/ttyS0", "Serial port to use.")

func main() {
	sigs := make(chan os.Signal)
	quit := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	c := &serial.Config{Name: *serialPort, Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Error opening serial port %v", err)
	}
	defer s.Close()

	newFileChan := make(chan bool)
	fileWriterChan := make(chan string)
	stopchan := make(chan struct{})
	stoppedchan := make(chan struct{})

	go readStdin(s, newFileChan)
	go readSerial(s, fileWriterChan)
	go fileWriter(newFileChan, fileWriterChan, stopchan, stoppedchan)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		quit <- true
	}()

	fmt.Println(">> started")
	<-quit
	close(stopchan)
	<-stoppedchan
	fmt.Println(">> exiting")
}
