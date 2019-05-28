package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Build new contest object
func BuildNewContest(name string, shortName string, durationHours int) (contest Contest) {
	nowTime := time.Now().Unix()
	activateTime := float64(nowTime) + 0.00
	startTime := activateTime + 10.00
	freezeTime := activateTime + 20.00
	endTime := startTime + float64(durationHours*3600)
	unfreezeTime := endTime + 10.00
	deactivateTime := activateTime + float64(60*86400) // deactivate after 2 months

	return Contest{
		Name:           name,
		ShortName:      shortName,
		ActivateTime:   activateTime,
		StartTime:      startTime,
		FreezeTime:     freezeTime,
		EndTime:        endTime,
		UnfreezeTime:   unfreezeTime,
		DeactivateTime: deactivateTime,

		ActivateTimeString:   time.Unix(int64(activateTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		StartTimeString:      time.Unix(int64(startTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		FreezeTimeString:     time.Unix(int64(freezeTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		EndTimeString:        time.Unix(int64(endTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		UnfreezeTimeString:   time.Unix(int64(unfreezeTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		DeactivateTimeString: time.Unix(int64(deactivateTime), 0).Format("2006-01-02 15:04:05 Asia/Kolkata"),
		Enabled:              1,
		Public:               0,

		DurationHours: durationHours,
	}
}

// Get contest from mysql db by short name
func GetContestByShortName(contestShortName string, config *Config) (contest Contest, err error) {
	// Get contest with greatest ID
	var curContests []Contest
	if err = config.Db.Table("contest").Limit(1).Where("shortname = ?", contestShortName).Find(&curContests).Error; err != nil {
		return contest, PrintErr("READ_CONTEST_BY_SHORTNAME_ERR", fmt.Sprintf("%v", err))
	}
	if len(curContests) > 0 {
		return curContests[0], nil
	}
	return contest, nil
}

// Create a new contest in contests table
// ContestId (cid column) is set by reading the latest contest from contest table and incrementing it by 1
func CreateContest(newContest Contest, config *Config) (err error) {
	var contests []Contest
	// Get contest with greatest ID
	if err = config.Db.Table("contest").Limit(1).Order("cid desc").Find(&contests).Error; err != nil {
		return PrintErr("READ_LATEST_CONTEST_ERR", fmt.Sprintf("%v", err))
	}
	if len(contests) == 0 {
		return PrintErr("READ_LATEST_CONTEST_ERR", "No contests read")
	}

	// Add index to email column of user table
	rows, err := config.Db.Raw("SELECT COUNT(1) IndexIsThere FROM INFORMATION_SCHEMA.STATISTICS WHERE table_schema=DATABASE() AND table_name='user' AND index_name='user_email';").Rows()
	if err != nil {
		return PrintErr("READ_EMAILINDEX_ERR", fmt.Sprintf("contestshortname: %s): %v", newContest.ShortName, err))
	}
	defer rows.Close()
	var result int
	for rows.Next() {
		rows.Scan(&result)
		if result == 0 {
			log.Printf("EMAILINDEX: No email index found for user table, creating one: %d\n", result)
			if err = config.Db.Exec("CREATE INDEX user_email ON user (email) USING BTREE;").Error; err != nil {
				return PrintErr("CREATE_EMAIL_INDEX_ERROR", fmt.Sprintf("Error creating index on email column of user table: %v", err))
			}
		} else {
			log.Printf("EMAILINDEX_ALREADYPRESENT\n")
		}
	}

	// Check if contest already created
	curContest, err := GetContestByShortName(newContest.ShortName, config)
	if err != nil {
		return err
	}
	if curContest.Name != "" && curContest.Cid > 0 {
		log.Printf("CONTEST_ALREADY_PRESENT: (shortname: %s, fullname: %s)\n", newContest.ShortName, curContest.Name)
		return nil
	}

	newCid := contests[0].Cid + 1
	newContest.Cid = newCid
	PrintVal("LATEST_CONTEST", contests)
	PrintVal("NEW_CONTEST", newContest)
	if err = config.Db.Table("contest").Create(newContest).Error; err != nil {
		return PrintErr("INSERT_CONTEST_TABLE_ERR", fmt.Sprintf("Error inserting %s into 'contest' table: %v", newContest.ShortName, err))
	}
	return nil
}

// Delete contest by its short-name with all its users, teams and submissions
func DeleteContestFull(contestShortName string, config *Config) (err error) {
	if contestShortName == "" {
		return PrintErr("DELETE_EMPTY_SHORT_NAME", "")
	}

	var contests []Contest
	// Get contest with greatest ID
	if err = config.Db.Table("contest").Limit(1).Where("shortname = ?", contestShortName).Find(&contests).Error; err != nil {
		return PrintErr("READ_CONTEST_ERR", fmt.Sprintf("failed to read %s: %v", contestShortName, err))
	}
	if len(contests) == 0 {
		return PrintErr("CONTEST_NOT_FOUND_TO_DELETE", fmt.Sprintf("No contest found for %s", contestShortName))
	}
	PrintVal("CONTEST", contests)
	contestId := contests[0].Cid

	// Find teams which have been registered for the contest
	rows, err := config.Db.Table("contestteam").Raw("SELECT teamid from contestteam WHERE cid = ?", contestId).Rows()
	if err != nil {
		return PrintErr("READ_CONTESTTEAMS_ERR", fmt.Sprintf("contestshortname: %s, contestid: %d): %v", contestShortName, contestId, err))
	}
	defer rows.Close()
	var teamId int
	for rows.Next() {
		rows.Scan(&teamId)
		log.Printf("DELETE_TEAMID: Deleting teamId: %v\n", teamId)
		DeleteUser("userid", teamId, contestId, config)
	}

	log.Printf("DELETE_FROM_CONTEST_TABLE: Deleting %s from 'contest' table\n", contestShortName)
	if err = config.Db.Table("contest").Delete(Contest{}, "shortname = ?", contestShortName).Error; err != nil {
		return PrintErr("DELETE_FROM_CONTEST_TABLE_ERR", fmt.Sprintf("Error deleting %s from 'contest' table: %v", contestShortName, err))
	}

	return nil
}

// Fetch contest results from database
func FetchResults(contestShortName string, config *Config) (users []*User, teamScores []*TeamScore, err error) {
	curContest, err := GetContestByShortName(contestShortName, config)
	if err != nil {
		return nil, nil, PrintErr("CONTEST_FETCH_ERR", fmt.Sprintf("contestshortname: %s): %v", contestShortName, err))
	}
	if curContest.Name == "" || curContest.Cid == 0 {
		return nil, nil, PrintErr("CONTEST_NOT_FOUND", fmt.Sprintf("contestshortname: %s): %v", contestShortName, err))
	}

	sqlQuery := `SELECT cid, teamid, points_restricted, totaltime_restricted FROM rankcache WHERE cid = ? ORDER BY points_restricted DESC, totaltime_restricted ASC`
	rows, err := config.Db.Raw(sqlQuery, curContest.Cid).Rows()
	if err != nil {
		return nil, nil, PrintErr("SCOREBOARD_GET_ERR", fmt.Sprintf("contestshortname: %s): %v", contestShortName, err))
	}
	defer rows.Close()

	users = make([]*User, 0)
	teamScores = make([]*TeamScore, 0)
	log.Printf("TEAM_SCORE_FETCH: (contestid %d)\n", curContest.Cid)
	for rows.Next() {
		teamScore := new(TeamScore)
		err := rows.Scan(&teamScore.Cid, &teamScore.TeamId, &teamScore.Points, &teamScore.TimeTaken)
		if err != nil {
			return nil, nil, PrintErr("FETCH_SCORE_ERR", fmt.Sprintf("failed to fetch score (teamId %d): %v", teamScore.TeamId, err))
		}
		PrintVal("TEAM_SCORE", teamScore)
		user, err := GetUserById("userid", teamScore.TeamId, false, config.Db)
		if err != nil {
			return nil, nil, PrintErr("FETCH_USER_BY_ID_ERR", fmt.Sprintf("failed to fetch user (teamId %d): %v", teamScore.TeamId, err))
		}
		users = append(users, user)
		teamScores = append(teamScores, teamScore)
	}

	return users, teamScores, nil
}

// Fetch contest results and export as TSV
func ExportResultsTSV(contestShortName string, config *Config) (err error) {
	users, teamScores, err := FetchResults(contestShortName, config)
	if err != nil {
		return err
	}
	outputFilename := config.CliArgs.ResultsFile
	outputFile, err := os.OpenFile(outputFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return PrintErr("FILE_WOPEN_ERR", fmt.Sprintf("%s: %v", outputFilename, err))
	}
	defer outputFile.Close()
	text := fmt.Sprintf("email\tusername\tuserid\tcontestid\tpoints\ttotaltime\n")
	if _, err = outputFile.WriteString(text); err != nil {
		return PrintErr("RESULTS_HEADER_WRITE_ERR", fmt.Sprintf("failed to print header details to %s: %v\n", outputFilename, err))
	}
	for i := 0; i < len(users); i++ {
		line := fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\n", users[i].Email, users[i].Name, users[i].UserId, teamScores[i].Cid, teamScores[i].Points, teamScores[i].TimeTaken)
		if _, err = outputFile.WriteString(line); err != nil {
			return PrintErr("RESULTS_WRITE_ERR", fmt.Sprintf("failed to print results to %s: %v\n", outputFilename, err))
		}
	}
	return nil
}
