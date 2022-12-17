package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	HelpFlag   = "h"
	VerFlag    = "v"
	EditFlag   = "e"
	LogFlag    = "log"
	UpdateFlag = "u"
)

func main() {
	fs := flag.NewFlagSet("cheat-sheet flag set", flag.ContinueOnError)

	fs.Bool(VerFlag, false, "print version")
	fs.Bool(HelpFlag, false, "print usage")
	fs.Bool(LogFlag, false, "print log")
	fs.Bool(UpdateFlag, false, "update tldr cache")
	fs.String(EditFlag, "", "edit cheat-sheet name")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Printf("error ocurred: %v\n", err)
		return
	}

	if err := Run(fs); err != nil {
		fmt.Printf("error ocurred: %v\n", err)
	}
}

func Run(fs *flag.FlagSet) error {
	cmd := CreateCommand(fs)

	if cmd.PrintLog() {
		log.Printf("create a new command %+v\n", cmd)
	}

	cfg, err := DefaultConfig()
	if err != nil {
		return err
	}

	executor := NewExecutor(cfg)
	return executor.Exec(cmd)
}
