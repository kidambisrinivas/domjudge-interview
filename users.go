package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/jinzhu/gorm"
)

// Create users from tsv file full of emailIDs
// INPUT: filename of tsv file which has 1 column [Email ID of users]
// OUTPUT: filename of tsv file which has 4 columns [Email ID of users, userid, teamid, password]
func PerformOpOnFile(filename string, contestShortName string, op string, config *Config) (err error) {
	var contests []Contest
	// Get contest with greatest ID
	if err = config.Db.Table("contest").Limit(1).Where("shortname = ?", contestShortName).Find(&contests).Error; err != nil {
		return PrintErr("READ_CONTEST_ERR", fmt.Sprintf("%v", err))
	}
	if len(contests) == 0 {
		return PrintErr("CONTEST_NOT_FOUND_ERR", fmt.Sprintf("no contest found for %s", contestShortName))
	}
	PrintVal("CONTEST", contests)
	contestId := contests[0].Cid
	contestDetails := contests[0]

	file, err := os.Open(filename)
	if err != nil {
		return PrintErr("FILE_OPEN_ERR", fmt.Sprintf("%v", err))
	}
	defer file.Close()

	outputFilename := fmt.Sprintf("%s.details", filename)
	outputFile, err := os.OpenFile(outputFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return PrintErr("FILE_WOPEN_ERR", fmt.Sprintf("%s: %v", outputFilename, err))
	}
	defer outputFile.Close()
	text := fmt.Sprintf("email\tusername\tpassword\tteamid\n")
	if _, err = outputFile.WriteString(text); err != nil {
		return PrintErr("USERDETAILS_PRINT_ERR: failed to print user header details: %v\n", fmt.Sprintf("%v", err))
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("LINE_READ: (%s) Attempting to create user...\n", line)

		// Get user with email ID, if already present dont create a new one
		var user *User
		if strings.HasSuffix(op, "USERS") {
			user, err = GetUserById("email", line, false, config.Db)
			if err != nil && !strings.Contains(err.Error(), "USER_NOT_FOUND") {
				return PrintErr("READ_USER_BY_EMAIL_ERR", fmt.Sprintf("(email %s): %v", line, err))
			}
		}

		if op == "ADD_USERS" {
			if user != nil && user.Email != "" && user.UserId > 0 {
				log.Printf("USER_ALREADY_PRESENT: (%s) user already present, skipping ...\n", line)
			} else {
				newUser, err := CreateUser(line, contestId, config)
				if err == nil {
					text := fmt.Sprintf("%s\t%s\t%s\t%d\n", newUser.Email, newUser.Username, newUser.ClearPassword, newUser.TeamId)
					if _, err = outputFile.WriteString(text); err != nil {
						log.Printf("USERDETAILS_PRINT_ERR: failed to print user details for user (%v): %v\n", text, err)
					}

					// Send credentials by email
					SendContestWelcomeEmail(newUser, contestDetails, config)
				}
			}
		} else if op == "RESEND_EMAIL_USERS" {
			err = UpdateUserPassword(user, config)
			if err == nil {
				text := fmt.Sprintf("%s\t%s\t%s\t%d\n", user.Email, user.Username, user.ClearPassword, user.TeamId)
				if _, err = outputFile.WriteString(text); err != nil {
					log.Printf("USERDETAILS_PRINT_ERR: failed to print user details for user (%v): %v\n", text, err)
				}
				// Send credentials by email
				SendContestWelcomeEmail(*user, contestDetails, config)
			}
		} else if op == "DELETE_USERS" {
			DeleteUser("email", line, contestId, config)
		}
	}
	if err = scanner.Err(); err != nil {
		return PrintErr("FILE_READ_ERR", fmt.Sprintf("%v", err))
	}

	log.Printf("Finished %s users from file %s for contest %s\n", op, filename, contestShortName)
	return nil
}

// Get user from mysql db by userid
func GetUserById(field string, value interface{}, isTxn bool, db *gorm.DB) (user *User, err error) {
	var users []User
	// Get user with greatest ID
	sqlQuery := fmt.Sprintf("%s = ?", field)
	if err = db.Table("user").Limit(1).Where(sqlQuery, value).Find(&users).Error; err != nil {
		if isTxn {
			db.Rollback()
		}
		return user, PrintErr("READ_USER_BY_FIELD_ERR", fmt.Sprintf("%s: %v): %v", field, value, err))
	}
	if len(users) == 0 {
		if isTxn {
			db.Rollback()
		}
		return user, PrintErr("USER_NOT_FOUND", fmt.Sprintf("(%s: %v)", field, value))
	}
	user1 := users[0]
	user = &user1
	PrintVal("USER_FETCHED", user)
	return user, nil
}

// Build new user from emailId
func BuildNewUser(emailId string, newTeamId int) (user User, team Team, err error) {
	re := regexp.MustCompile(`\@.*`)
	name := re.ReplaceAllString(emailId, "")
	clearPassword := RandStringBytesMaskImprSrcUnsafe(10)
	username := fmt.Sprintf("user%d", newTeamId)
	hashPassword, err := GetPasswordHash(clearPassword)
	if err != nil {
		return user, team, err
	}
	user = User{
		UserId:        newTeamId,
		Username:      username,
		Name:          name,
		Email:         emailId,
		ClearPassword: clearPassword,
		HashPassword:  hashPassword,
		Enabled:       1,
		TeamId:        newTeamId,
	}
	team = Team{
		TeamId:     newTeamId,
		Name:       username,
		CategoryId: 3,
		Enabled:    1,
		Members:    username,
		Penalty:    0,
	}
	return user, team, nil
}

// Create a new user in DOMJudge
// 1. Creates a new team in team table (TeamId (teamid column) is set by reading the latest team from team table and incrementing it by 1)
// 2. Creates a new user in user table (UserId (userid column) is set by reading the latest user from user table and incrementing it by 1)
// 3. Inserts user into userrole table
// 4. Adds contest to the user team
func CreateUser(emailId string, contestId int, config *Config) (newUser User, err error) {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err = tx.Error; err != nil {
		return newUser, PrintErr("TXN_OBJ_ERR", fmt.Sprintf("%v", err))
	}

	// 1. Insert new team
	var teams []Team
	// Get team with greatest ID
	if err = tx.Table("team").Limit(1).Order("teamid desc").Find(&teams).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("READ_LATEST_TEAM_ERR", fmt.Sprintf("%v", err))
	}
	newTeamId := 0
	if len(teams) == 0 {
		newTeamId = 1
	}
	newTeamId = teams[0].TeamId + 1
	newUser, newTeam, err := BuildNewUser(emailId, newTeamId)
	if err != nil {
		tx.Rollback()
		return newUser, PrintErr("HASH_PASSWORD_ERR", fmt.Sprintf("%v", err))
	}
	PrintVal("LATEST_TEAM", teams)
	PrintVal("NEW_TEAM", newTeam)
	if err = tx.Table("team").Create(newTeam).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("INSERT_TEAM_TABLE_ERR", fmt.Sprintf("Error inserting %s into 'team' table: %v", newTeam.Name, err))
	}

	// 2. Insert new user
	var users []User
	// Get user with greatest ID
	if err = tx.Table("user").Limit(1).Order("userid desc").Find(&users).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("READ_LATEST_USER_ERR", fmt.Sprintf("%v", err))
	}
	PrintVal("LATEST_USER", users)
	PrintVal("NEW_USER", newUser)
	if err = tx.Table("user").Create(newUser).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("INSERT_USER_TABLE_ERR", fmt.Sprintf("Error inserting %s into 'user' table: %v", newUser.Email, err))
	}

	// 3. Insert new userrole
	var userroles []UserRole
	// Get user with greatest ID
	if err = tx.Table("userrole").Limit(1).Where("userid = ?", newUser.UserId).Find(&userroles).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("READ_USERROLE_ERR", fmt.Sprintf("%v", err))
	}
	PrintVal("USERROLE", userroles)
	if len(userroles) == 0 {
		newUserRole := UserRole{
			UserId: newUser.UserId,
			RoleId: 3,
		}
		PrintVal("NEW_USERROLE", newUserRole)
		if err = tx.Table("userrole").Create(newUserRole).Error; err != nil {
			tx.Rollback()
			return newUser, PrintErr("INSERT_USERROLE_TABLE_ERR", fmt.Sprintf("Error inserting %s into 'userrole' table: %v", newUser.Email, err))
		}
	}

	// 4. Add contest to team
	var contestTeams []ContestTeam
	// Get user with greatest ID
	if err = tx.Table("contestteam").Limit(1).Where("teamid = ?", newTeam.TeamId).Find(&contestTeams).Error; err != nil {
		tx.Rollback()
		return newUser, PrintErr("READ_CONTESTTEAM_ERR", fmt.Sprintf("%v", err))
	}
	PrintVal("CONTESTTEAMS", userroles)
	if len(contestTeams) == 0 {
		newContestTeam := ContestTeam{
			Cid:    contestId,
			TeamId: newTeam.TeamId,
		}
		PrintVal("NEW_CONTESTTEAM", newContestTeam)
		if err = tx.Table("contestteam").Create(newContestTeam).Error; err != nil {
			tx.Rollback()
			return newUser, PrintErr("INSERT_CONTESTTEAM_TABLE_ERR", fmt.Sprintf("Error inserting %s into 'contestteam' table: %v", newUser.Email, err))
		}
	}

	if err = tx.Commit().Error; err != nil {
		return newUser, PrintErr("TX_COMMIT_ERR", fmt.Sprintf("Error inserting %s into tables as txn: %v", newUser.Email, err))
	}
	return newUser, nil
}

// Update user's password in database
func UpdateUserPassword(user *User, config *Config) (err error) {
	newUser, _, err := BuildNewUser(user.Email, user.TeamId)
	if err != nil {
		return err
	}
	user.ClearPassword = newUser.ClearPassword
	user.HashPassword = newUser.HashPassword
	if err = config.Db.Table("user").Model(user).Updates(map[string]interface{}{"password": user.HashPassword}).Error; err != nil {
		return PrintErr("UPDATE_PASSWORD_ERR", fmt.Sprintf("Error updating %s 'user': %v", user.Email, err))
	}
	return nil
}

// Delete a user from DOMJudge
// 4. Delete contest from user team
// 3. Delete user from userrole table
// 2. Delete user in user table
// 1. Delete team in team table
func DeleteUser(field string, value interface{}, contestId int, config *Config) (err error) {
	tx := config.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err = tx.Error; err != nil {
		return PrintErr("TXN_OBJ_ERR", fmt.Sprintf("%v", err))
	}

	// 0. Read user and team
	var user User
	var team Team
	var users []User
	// Get user with greatest ID
	sqlQuery := fmt.Sprintf("%s = ?", field)
	if err = tx.Table("user").Limit(1).Where(sqlQuery, value).Find(&users).Error; err != nil {
		tx.Rollback()
		return PrintErr("READ_USER_BY_EMAIL_ERR", fmt.Sprintf("%s: %s, contestid: %d): %v", field, value, contestId, err))
	}
	if len(users) == 0 {
		tx.Rollback()
		return PrintErr("NO_USER_TO_DELETE", fmt.Sprintf("%s: %s, contestid: %d): %v", field, value, contestId, err))
	}
	user = users[0]
	PrintVal("USER_TO_DELETE", user)
	var teams []Team
	// Get team with greatest ID
	if err = tx.Table("team").Where("teamid = ?", user.TeamId).Find(&teams).Error; err != nil {
		tx.Rollback()
		return PrintErr("READ_TEAM_ERR", fmt.Sprintf("email: %s, contestid: %d): %v", user.Email, contestId, err))
	}
	team = teams[0]
	PrintVal("TEAM_TO_DELETE", team)

	// 4. Delete contest from user team
	if err = tx.Table("contestteam").Delete(ContestTeam{}, "teamid = ?", team.TeamId).Error; err != nil {
		tx.Rollback()
		return PrintErr("DELETE_CONTESTTEAM_ERR", fmt.Sprintf("email: %s, username: %s, teamid: %d, contestid: %d): %v", user.Email, user.Username, user.TeamId, contestId, err))
	}
	log.Printf("DELETE_CONTESTTEAM_SUCCESS: (email: %s, username: %s, teamid: %d, contestid: %d)\n", user.Email, user.Username, user.TeamId, contestId)

	// 3. Delete user from userrole table
	if err = tx.Table("userrole").Delete(UserRole{}, "userid = ?", user.UserId).Error; err != nil {
		tx.Rollback()
		return PrintErr("DELETE_USERROLE_ERR", fmt.Sprintf("email: %s, username: %s, teamid: %d, contestid: %d): %v", user.Email, user.Username, user.TeamId, contestId, err))
	}
	log.Printf("DELETE_USERROLE_SUCCESS: (email: %s, username: %s, teamid: %d, contestid: %d)\n", user.Email, user.Username, user.TeamId, contestId)

	// 2. Delete user in user table
	if err = tx.Table("user").Delete(User{}, "userid = ?", user.UserId).Error; err != nil {
		tx.Rollback()
		return PrintErr("DELETE_USER_ERR", fmt.Sprintf("email: %s, username: %s, teamid: %d, contestid: %d): %v", user.Email, user.Username, user.TeamId, contestId, err))
	}
	log.Printf("DELETE_USER_SUCCESS: (email: %s, username: %s, teamid: %d, contestid: %d)\n", user.Email, user.Username, user.TeamId, contestId)

	// 1. Delete team in team table
	if err = tx.Table("team").Delete(User{}, "teamid = ?", user.TeamId).Error; err != nil {
		tx.Rollback()
		return PrintErr("DELETE_TEAM_ERR", fmt.Sprintf("email: %s, username: %s, teamid: %d, contestid: %d): %v", user.Email, user.Username, user.TeamId, contestId, err))
	}
	log.Printf("DELETE_TEAM_SUCCESS: (email: %s, username: %s, teamid: %d, contestid: %d)\n", user.Email, user.Username, user.TeamId, contestId)

	if err = tx.Commit().Error; err != nil {
		return PrintErr("TX_COMMIT_ERR", fmt.Sprintf("Error deleting %s from tables as txn: %v", user.Email, err))
	}
	return nil
}

// Send contest welcome email to a user
func SendContestWelcomeEmail(user User, contestDetails Contest, config *Config) (err error) {
	to := user.Email
	toName := user.Name
	from := config.CliArgs.SendwithusFrom
	fromName := config.CliArgs.SendwithusFromName
	replyTo := config.CliArgs.SendwithusReplyTo
	cc := ""  // comma separated email id's
	bcc := "" // comma separated email id's
	sendwithusApiKey := config.CliArgs.SendwithusApiKey
	sendwithusTemplateId := config.CliArgs.SendwithusTemplateId
	templateData := &ContestWelcomeEmail{
		ContestUrl:       config.CliArgs.ContestUrl,
		Deadline:         contestDetails.EndTimeString,
		FirstName:        user.Name,
		Title:            contestDetails.Name,
		Username:         user.Username,
		Password:         user.ClearPassword,
		ContestShortName: contestDetails.ShortName,
		FromName:         fromName,
	}
	_, err = SendEmailUsingSendwithus(to, toName, from, fromName, replyTo, cc, bcc, sendwithusApiKey, sendwithusTemplateId, templateData)
	return err
}
