# DOMJudge Contest Setup

domjudge-interview helps setup a contest in a running DOMJudge server with MYSQL database and manage large number of users typically for usecases like `Conducting Interviews`. It will help you create 100s of users from a TSV file full of email IDs with 1 member per team (associated with the user) by performing SQL queries to DOMJudge MySql database.

This service supports the following operations:

* `CREATE_CONTEST`: Create a contest by name and set activate, start times in DOMJudge database
* `ADD_USERS`: Add users by email ID from a file to the DOMJudge database and add then to a contest identified by contest-short-name
* `DELETE_USERS`: Delete users by email ID from a file to the DOMJudge database and remove them from a contest identified by contest-short-name
* `DELETE_CONTEST`: Delete contest and all teams and users associated with that contest
* `SHOW_RESULTS`: Export leaderboard (Results) of a contest identified by contest-short-name to a TSV file 

## Service modes

### `CREATE_CONTEST`

This service mode creates a contest by performing the following SQL queries to DOMJudge database:

- SQL Query 1: Read latest contest id using: `SELECT cid, name, shortname FROM contest ORDER BY cid DESC LIMIT 10;`
- Set contest end time to be 2 days after start time and deactivetime to be 1 month after activate time
- Freeze time should be set to 5 mins after contest start time
- SQL Query 2: Create contest with id as latestCid+1 using insert command above (for Create contests)
  * Sample SQL query: `INSERT INTO contest (cid, name, shortname, activatetime, starttime, freezetime, endtime, unfreezetime, deactivatetime, activatetime_string, starttime_string, freezetime_string, endtime_string, unfreezetime_string, deactivatetime_string, public) VALUES (2, "May 2019 Interview", "int-28-may", 1558422000, 1558422000, 1558422000, 1558422000, 1558422000, 1558422000, "2019-05-21 12:00:00 Asia/Kolkata", "2019-05-21 12:00:00 Asia/Kolkata", "2019-05-21 12:00:00 Asia/Kolkata", "2019-05-21 12:00:00 Asia/Kolkata", "2019-05-21 12:00:00 Asia/Kolkata", "2019-05-21 12:00:00 Asia/Kolkata", 0)`

```bash
export DB_CONN_STR="user:pass@tcp(db-host:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
$GOPATH/bin/domjudge-interview --op CREATE_CONTEST --contest-name "Full Stack Engineer" --contest-short-name fs-1-may-2019 --contest-duration-hours 48 --db-conn-str "$DB_CONN_STR"
```

### `ADD_USERS`

This service mode add users (by email addresses) from a file to DOMJudge database

- Read latest team id using: `SELECT teamid FROM team ORDER BY teamid DESC LIMIT 10;`
- Add users by emailid starting from userid+1
- Generate password for each user
- Insert new user into DOMJudge database
  1. Add team first
  	- Sample SQL query: `INSERT INTO team (teamid, name,categoryid,members) VALUES (28, "user1", 3, "user1");`
  	- [Source code](https://github.com/DOMjudge/domjudge/blob/master/misc-tools/create_accounts.in#L36)
  2. Add user next
  	- Sample SQL query: `INSERT INTO user (userid,username,name,email,password,teamid) VALUES (28,"user1","user1","user1@gmail.com","$2a$10$RR/lyfRhrlL0ngq7vFdPnuwbh44YXsOZ2yqVwD.Ns/5zR/Xm0vpfm",28);`
  	- [Source code](https://github.com/DOMjudge/domjudge/blob/master/misc-tools/create_accounts.in#L36)
  3. Add userrole next
  	- Sample SQL query: `INSERT INTO userrole (userid, roleid) VALUES (28, 3);`
  4. Add contests to teams finally
  	- Sample SQL query: `INSERT INTO contestteam (cid, teamid) VALUES (1, 28);`
- INPUT: file with emailids (1 column), OUTPUT: file with emailids, userids, passwords (3 columns)

#### Password generation

DOMJudge Database Reference

- [Password generation source](https://github.com/DOMjudge/domjudge/blob/master/lib/lib.wrappers.php#L84)
- [PHP password hash function](https://www.php.net/manual/en/function.password-hash.php)
- [PASSWORD_HASH_COST](https://github.com/DOMjudge/domjudge/blob/master/etc/domserver-config.php#L7)

```bash
export DB_CONN_STR="user:pass@tcp(db-host:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
$GOPATH/bin/domjudge-interview --op ADD_USERS --contest-short-name fs-1-may-2019 --users-file "user_emails.tsv" --db-conn-str "$DB_CONN_STR" --sendwithus-api-key "$APIKEY" --sendwithus-template-id "tem_sdfq345" --sendwithus-reply-to "hiring@mycompany.com" --sendwithus-from "hiring@mycompany.com" --sendwithus-from-name "YOUR_NAME" --contest-url "https://mycompany.com/contest/login"
```

contest-url link above is the DOMJudge web UI link

For ease of use, you could use the following config file way of invoking the above command

```bash
$GOPATH/bin/domjudge-interview --op ADD_USERS --contest-short-name fs-1-may-2019 --users-file "user_emails.tsv" --config .domjudge-interview.json
```

### `DELETE_USERS`

Delete users by email id from DOMJudge database. This mode will find users by email ID from user
table and delete the user from the following tables

* user
* team correspoding user in team table
* user from userrole
* team from contestteam

```bash
export DB_CONN_STR="user:pass@tcp(db-host:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
$GOPATH/bin/domjudge-interview --op DELETE_USERS --contest-short-name fs-1-may-2019 --users-file "user_emails.tsv" --db-conn-str "$DB_CONN_STR"
```

### `DELETE_CONTEST`

Delete contest with all its users, teams and entries in userrole, contestteam tables.

* for every team with access to the contest (from contestteam), delete
  * user
  * team correspoding user in team table
  * user from userrole
  * team from contestteam
* delete contest from contest table

```bash
export DB_CONN_STR="user:pass@tcp(db-host:3306)/dbname?charset=utf8&parseTime=True&loc=Local"
$GOPATH/bin/domjudge-interview --op DELETE_CONTEST --contest-short-name fs-1-may-2019 --db-conn-str "$DB_CONN_STR"
```

### `SHOW_RESULTS`

- Show results of contests reverse sorted by points and score
- OUTPUT: file with emailids, userids, points, totaltime

```bash
$GOPATH/bin/domjudge-interview --op SHOW_RESULTS --contest-short-name 11-apr --results-file "$HOME/seedFiles/apr11.results.tsv" --db-conn-str "$DB_CONN_STR2"
```

## Config file format

All of the above command line parameters can be stored in a config file which can just be passed
to this binary for easy usage of this service. 

```javascript
{
	"contest-name": "Full stack engineer",
	"contest-short-name": "fs-1-may-2019",
	"contest-duration-hours": 48,
	"users-file": "$HOME/domjudge_c1_users.tsv",
	"results-file": "$HOME/apr11.results.tsv",
	"db-conn-str": "user:pswd@tcp(db_host:3306)/db_name?charset=utf8&parseTime=True&loc=Local",
	"sendwithus-api-key": "live_myapikey",
	"sendwithus-template-id": "tem_mytemplatekey",
	"sendwithus-reply-to": "contest@mycompany.com",
	"sendwithus-from": "contest@company.com",
	"sendwithus-from-name": "MyCompany Hiring Team",
	"contest-url": "https://mycompany.com/contest/login"
}
```
