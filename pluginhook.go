package main

import (
	"os"
	"flag"
	"fmt"
	"os/exec"
	"syscall"
	"path/filepath"
	"log"
	"code.google.com/p/go.crypto/ssh/terminal"
)

func main() {

	var serial = flag.Bool("serial", false, "Run plugins in series rather than in parallel")
	flag.Parse()

	pluginPath := os.Getenv("PLUGIN_PATH") 
	if pluginPath == "" {
		log.Fatal("[ERROR] Unable to locate plugins: set $PLUGIN_PATH\n")
		os.Exit(1)
	}
	if len(flag.Args()) < 1 {
		log.Fatal("[ERROR] Hook name argument is required\n")
		os.Exit(1)
	}
	cmds := make([]exec.Cmd, 0)
	var matches, _ = filepath.Glob(fmt.Sprintf("%s/*/%s", pluginPath, flag.Args()[0]))
	for _, hook := range matches {
		cmd := exec.Command(hook, os.Args[2:]...)
		cmds = append(cmds, *cmd)
	}
	done := make(chan bool, len(cmds))

	serial_mutex := make(chan bool, len(cmds))

	for i := len(cmds)-1; i >= 0; i-- {
		if (! *serial) {
			if i == len(cmds)-1 {
				cmds[i].Stdout = os.Stdout
			} 
			if i > 0 {
				stdout, err := cmds[i-1].StdoutPipe()
				if err != nil {
					log.Fatal(err)
				}
				cmds[i].Stdin = stdout
			}
			if i == 0 && !terminal.IsTerminal(syscall.Stdin) {
				cmds[i].Stdin = os.Stdin
			}
		} else {
			cmds[i].Stdout = os.Stdout
		}
		

		go func(cmd exec.Cmd) {

			if (*serial) {
				<- serial_mutex
			}

			err := cmd.Run()
			if msg, ok := err.(*exec.ExitError); ok { // there is error code 
				os.Exit(msg.Sys().(syscall.WaitStatus).ExitStatus())
			}

			done <- true
			serial_mutex <- true
		}(cmds[i])
	}

	if (*serial) {
		serial_mutex <- true
	}

	for i := 0; i < len(cmds); i++ {
		<-done
	}
}