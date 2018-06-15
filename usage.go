package main

import (
	"flag"
	"fmt"
	"os"
	"kuaishangtong/common/utils/log"
)

var _flags AppFlags

type AppFlags struct {
	Help 	bool
	Config 	string
}

func usage() {
	if _flags.Help {
		fmt.Printf("\nusage:\n")
		flag.PrintDefaults()
	}
}

func initFlags() bool {

	flag.StringVar(&_flags.Config, "c", "./conf.json", "specify config file")
	flag.BoolVar(&_flags.Help, "h", false, "print this message")

	flag.Usage = usage
	flag.Parse()

	if _flags.Help {
		usage()
		return false
	}

	if !exist(_flags.Config) {
		log.Errorf("can not find config file: %s", _flags.Config)
		return false
	}

	return true
}

func exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}