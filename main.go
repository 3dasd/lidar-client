package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tarm/serial"
)

func readSerial(p *serial.Port) {
	f, err := os.Create("/tmp/points")
	if err != nil {
		log.Fatalf("unable to open output file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		t := scanner.Text()

		if strings.HasPrefix(t, "p") {
			f.WriteString(t[1:])
			f.WriteString("\n")
		} else {
			fmt.Printf("> %s\n", t)
		}
	}
}

func readStdin(p *serial.Port) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		t := scanner.Text()
		p.Write([]byte(t))
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	quit := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("starting...")

	c := &serial.Config{Name: "/dev/ttyS0", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("Error opening serial port %v", err)
	}
	defer s.Close()

	go readSerial(s)
	go readStdin(s)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		quit <- true
	}()

	<-quit
	fmt.Println("exiting")
}
