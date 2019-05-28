package main

import (
	"log"
	"os"
)

func main() {

	// str1 := RandStringBytesMaskImprSrcUnsafe(10)
	// str2 := RandStringBytesMaskImprSrcUnsafe(10)
	// log.Printf("Str1: %s, Str2: %s\n", str1, str2)
	// os.Exit(1)

	config, err := NewConfig()
	if err != nil {
		os.Exit(1)
	}

	switch config.CliArgs.Op {
	case "CREATE_CONTEST":
		newContest := BuildNewContest(config.CliArgs.ContestName, config.CliArgs.ContestShortName, config.CliArgs.ContestDurationHours)
		err = CreateContest(newContest, config)
	case "ADD_USERS":
		err = PerformOpOnFile(config.CliArgs.UsersFile, config.CliArgs.ContestShortName, config.CliArgs.Op, config)
	case "RESEND_EMAIL_USERS":
		err = PerformOpOnFile(config.CliArgs.UsersFile, config.CliArgs.ContestShortName, config.CliArgs.Op, config)
	case "DELETE_CONTEST":
		err = DeleteContestFull(config.CliArgs.ContestShortName, config)
	case "DELETE_USERS":
		err = PerformOpOnFile(config.CliArgs.UsersFile, config.CliArgs.ContestShortName, config.CliArgs.Op, config)
	case "SHOW_RESULTS":
		err = ExportResultsTSV(config.CliArgs.ContestShortName, config)
	}
	if err != nil {
		log.Printf("MAIN_ERR: failed to perform (op %s, contest %s): %v", config.CliArgs.Op, config.CliArgs.ContestShortName, err)
	}
}
