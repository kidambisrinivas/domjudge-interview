package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func PrintVal(code string, val interface{}) {
	jsonStr, _ := json.MarshalIndent(val, "", "  ")
	log.Printf("%s: %s\n", code, string(jsonStr))
}

func PrintErr(code string, msg string) error {
	pmsg := fmt.Sprintf("%s: %s\n", code, msg)
	log.Printf(pmsg)
	return fmt.Errorf(pmsg)
}

// Validate if configuration details have been provided correctly for this service
func ValidateConfig(cliArgs *CliArgs) (err error) {
	if cliArgs.Op == "" {
		return PrintErr("CLI_ARG_ERR", "op arg missing")
	}
	if cliArgs.DbConnStr == "" {
		return PrintErr("CLI_ARG_ERR", "db-conn-str arg missing")
	}
	if cliArgs.ContestShortName == "" {
		return PrintErr("CLI_ARG_ERR", "contest-short-name arg missing")
	}

	switch cliArgs.Op {
	case "CREATE_CONTEST":
		if cliArgs.ContestName == "" || cliArgs.ContestDurationHours == 0 {
			if cliArgs.ContestName == "" {
				return PrintErr("CLI_ARG_ERR", "contest-name arg missing")
			}
			if cliArgs.ContestDurationHours == 0 {
				return PrintErr("CLI_ARG_ERR", "contest-duration-hours arg missing")
			}
		}
	case "ADD_USERS":
		if cliArgs.UsersFile == "" {
			return PrintErr("CLI_ARG_ERR", "users-file arg missing")
		}
		if _, err = os.Stat(cliArgs.UsersFile); os.IsNotExist(err) {
			return PrintErr("USER_FILE_NOT_EXIST", fmt.Sprintf("user-file arg file not found: %v", err))
		}
		if cliArgs.SendwithusApiKey != "" {
			if cliArgs.SendwithusReplyTo == "" || cliArgs.SendwithusTemplateId == "" || cliArgs.SendwithusFrom == "" || cliArgs.SendwithusFromName == "" || cliArgs.ContestUrl == "" {
				return PrintErr("SENDWITHUS_DETAILS_MISSING",
					fmt.Sprintf("if sendwithus-api-key is set, then both sendwithus-template-id, sendwithus-reply-to, sendwithus-from, contest-url and sendwithus-from-name must be present"))
			}
		}
	case "DELETE_CONTEST":
	case "DELETE_USERS":
		if cliArgs.UsersFile == "" {
			return PrintErr("CLI_ARG_ERR", "user-file arg missing")
		}
		if _, err = os.Stat(cliArgs.UsersFile); os.IsNotExist(err) {
			return PrintErr("USER_FILE_NOT_EXIST", fmt.Sprintf("user-file arg file not found: %v", err))
		}
	case "SHOW_RESULTS":
		if cliArgs.ResultsFile == "" {
			return PrintErr("CLI_ARG_ERR", "results-file arg missing")
		}
	}
	return nil
}

// Parse config file
func ParseConfigFile(filename string) (cliArgs *CliArgs, err error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, PrintErr("CLI_ARG_ERR", fmt.Sprintf("failed to read file %s: %v", filename, err))
	}
	cliArgs = new(CliArgs)
	err = json.Unmarshal(dat, cliArgs)
	if err != nil {
		return nil, PrintErr("CLI_ARG_ERR", fmt.Sprintf("failed to parse config file %s: %v", filename, err))
	}
	return cliArgs, nil
}

// Parse cli args
func ParseCliArgs() (cliArgs *CliArgs, err error) {
	config := flag.String("config", "", "Config file (OPTIONAL: For ease of use)")
	op := flag.String("op", "", "Which operation to perform (MANDATORY)")
	contestName := flag.String("contest-name", "", "Contest name (MANDATORY for op's: CREATE_CONTEST)")
	contestShortName := flag.String("contest-short-name", "", "Contest short name (MANDATORY)")
	contestDurationHours := flag.Int("contest-duration-hours", 0, "Contest duration hours (MANDATORY for op's: CREATE_CONTEST, START_CONTEST)")
	userFile := flag.String("users-file", "", "Users file to add users by email_id (MANDATORY for op's: ADD_USERS)")
	resultsFile := flag.String("results-file", "", "Results file to output contest results to (MANDATORY for op's: SHOW_RESULTS)")
	dbConnStr := flag.String("db-conn-str", "", "Mysql db to connect to create users (MANDATORY)")
	sendwithusApiKey := flag.String("sendwithus-api-key", "", "Sendwithus api key to send userid/password emails using sendwithus to all users (OPTIONAL for op ADD_USERS)")
	sendwithusTemplateId := flag.String("sendwithus-template-id", "", "Sendwithus template id to send userid/password emails using sendwithus to all users (OPTIONAL for op ADD_USERS)")
	sendwithusReplyTo := flag.String("sendwithus-reply-to", "", "Sendwithus reply to value to send userid/password emails using sendwithus to all users (OPTIONAL for op ADD_USERS, but MANDATORY if sendwithusApiKey is mentioned)")
	sendwithusFrom := flag.String("sendwithus-from", "", "Sendwithus from value to send userid/password emails using sendwithus to all users (OPTIONAL for op ADD_USERS, but MANDATORY if sendwithusApiKey is mentioned)")
	sendwithusFromName := flag.String("sendwithus-from-name", "", "Sendwithus from-name value to send userid/password emails using sendwithus to all users (OPTIONAL for op ADD_USERS, but MANDATORY if sendwithusApiKey is mentioned)")
	contestUrl := flag.String("contest-url", "", "Contest URL (MANDATORY for op's: ADD_USERS)")

	flag.Parse()

	cliArgs = &CliArgs{}
	if *config != "" {
		cliArgs, err = ParseConfigFile(*config)
		if err != nil {
			return nil, err
		}
	}
	cliArgs = &CliArgs{
		Op:                   getLastStr(cliArgs.Op, *op),
		ContestName:          getLastStr(cliArgs.ContestName, *contestName),
		ContestShortName:     getLastStr(cliArgs.ContestShortName, *contestShortName),
		ContestDurationHours: getLastInt(cliArgs.ContestDurationHours, *contestDurationHours),
		UsersFile:            getLastStr(cliArgs.UsersFile, *userFile),
		ResultsFile:          getLastStr(cliArgs.ResultsFile, *resultsFile),
		DbConnStr:            getLastStr(cliArgs.DbConnStr, *dbConnStr),
		SendwithusApiKey:     getLastStr(cliArgs.SendwithusApiKey, *sendwithusApiKey),
		SendwithusTemplateId: getLastStr(cliArgs.SendwithusTemplateId, *sendwithusTemplateId),
		SendwithusReplyTo:    getLastStr(cliArgs.SendwithusReplyTo, *sendwithusReplyTo),
		SendwithusFrom:       getLastStr(cliArgs.SendwithusFrom, *sendwithusFrom),
		SendwithusFromName:   getLastStr(cliArgs.SendwithusFromName, *sendwithusFromName),
		ContestUrl:           getLastStr(cliArgs.ContestUrl, *contestUrl),
	}
	err = ValidateConfig(cliArgs)
	return cliArgs, err
}

func getLastStr(str1 string, str2 string) string {
	if str2 != "" {
		return str2
	}
	return str1
}

func getLastInt(v1 int, v2 int) int {
	if v2 != 0 {
		return v2
	}
	return v1
}

func NewConfig() (config *Config, err error) {
	cliArgs, err := ParseCliArgs()
	if err != nil {
		return nil, err
	}

	// dbConnStr := "domjudge:djpw@domjudge-db.c97ivjugwy4b.us-east-1.rds.amazonaws.com:3306/domjudge_interview?charset=utf8&parseTime=True&loc=Local"
	dbConnStr := cliArgs.DbConnStr
	db, err := gorm.Open("mysql", dbConnStr)
	if err != nil {
		return nil, PrintErr("DB_CONN_ERR", fmt.Sprintf("Could not connect to %s: %v", dbConnStr, err))
	}
	config = &Config{
		CliArgs: cliArgs,
		Db:      db,
	}
	return config, nil
}
