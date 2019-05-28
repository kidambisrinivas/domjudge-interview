package main

import "github.com/jinzhu/gorm"

// Command line arguments to control this service
// Supported values for op: CREATE_CONTEST, ADD_USERS, DELETE_USERS, SHOW_RESULTS, START_CONTEST, END_CONTEST, FREEZE_CONTEST, UNFREEZE_CONTEST
type CliArgs struct {
	Op                   string `json:"op"`
	ContestName          string `json:"contest-name"`
	ContestShortName     string `json:"contest-short-name"`
	ContestDurationHours int    `json:"contest-duration-hours"`
	UsersFile            string `json:"users-file"`
	ResultsFile          string `json:"results-file"`
	DbConnStr            string `json:"db-conn-str"`
	SendwithusApiKey     string `json:"sendwithus-api-key"`
	SendwithusTemplateId string `json:"sendwithus-template-id"`
	SendwithusReplyTo    string `json:"sendwithus-reply-to"`
	SendwithusFrom       string `json:"sendwithus-from"`
	SendwithusFromName   string `json:"sendwithus-from-name"`
	ContestUrl           string `json:"contest-url"`
}

type Config struct {
	CliArgs *CliArgs `json:"cli_args"`
	Db      *gorm.DB `json:"db"`
}

type Contest struct {
	Cid                  int     `json:"cid" gorm:"column:cid;PRIMARY_KEY;"`
	ExternalId           string  `json:"externalid" gorm:"column:externalid;UNIQUE;"`
	Name                 string  `json:"contest-name" gorm:"column:name;"`
	ShortName            string  `json:"contest-short-name" gorm:"column:shortname;"`
	ActivateTime         float64 `json:"activatetime" gorm:"column:activatetime;"`
	StartTime            float64 `json:"starttime" gorm:"column:starttime;"`
	FreezeTime           float64 `json:"freezetime" gorm:"column:freezetime;"`
	UnfreezeTime         float64 `json:"unfreezetime" gorm:"column:unfreezetime;"`
	EndTime              float64 `json:"endtime" gorm:"column:endtime;"`
	DeactivateTime       float64 `json:"deactivatetime" gorm:"column:deactivatetime;"`
	ActivateTimeString   string  `json:"activatetime_string" gorm:"column:activatetime_string;"`
	StartTimeString      string  `json:"starttime_string" gorm:"column:starttime_string;"`
	FreezeTimeString     string  `json:"freezetime_string" gorm:"column:freezetime_string;"`
	UnfreezeTimeString   string  `json:"unfreezetime_string" gorm:"column:unfreezetime_string;"`
	EndTimeString        string  `json:"endtime_string" gorm:"column:endtime_string;"`
	DeactivateTimeString string  `json:"deactivatetime_string" gorm:"column:deactivatetime_string;"`
	Enabled              int     `json:"enabled" gorm:"column:enabled;"`
	Public               int     `json:"public" gorm:"column:public;"`

	DurationHours int `json:"contest-duration-hours" gorm:"-"`
}

type Team struct {
	TeamId             int      `json:"teamid" gorm:"column:teamid;PRIMARY_KEY;"`
	ExternalId         *string  `json:"externalid" gorm:"column:externalid;UNIQUE;"`
	Name               string   `json:"name" gorm:"column:name;"`
	CategoryId         int      `json:"categoryid" gorm:"column:categoryid;"`
	AffilId            *int     `json:"affilid" gorm:"column:affilid;"`
	Enabled            int      `json:"enabled" gorm:"column:enabled;"`
	Members            string   `json:"members" gorm:"column:members;type:longtext;"`
	Room               *string  `json:"room" gorm:"column:room"`
	comments           *string  `json:"comments" gorm:"column:comments"`
	JudgingLastStarted *float64 `json:"judging_last_started" gorm:"column:judging_last_started"`
	Penalty            int      `json:"penalty" gorm:"column:penalty"`
}

type User struct {
	UserId        int      `json:"userid" gorm:"column:userid;PRIMARY_KEY;"`
	Username      string   `json:"username" gorm:"column:username;UNIQUE"`
	Name          string   `json:"name" gorm:"column:name;"`
	Email         string   `json:"email" gorm:"column:email;"`
	LastLogin     *float64 `json:"last_login" gorm:"column:last_login;"`
	LastIpAddress *string  `json:"last_ip_address" gorm:"column:last_ip_address;"`
	ClearPassword string   `json:"clear_password" gorm:"-"`
	HashPassword  string   `json:"hash_password" gorm:"column:password;"`
	IpAddress     *string  `json:"ip_address" gorm:"column:ip_address;"`
	Enabled       int      `json:"enabled" gorm:"column:enabled"`
	TeamId        int      `json:"teamid" gorm:"column:teamid"`
}

type UserRole struct {
	UserId int `json:"userid" gorm:"column:userid;"`
	RoleId int `json:"roleid" gorm:"column:roleid;"`
}

type TeamScore struct {
	Cid       int   `json:"cid" gorm:"column:cid;"`
	TeamId    int   `json:"teamid" gorm:"column:teamid;"`
	Points    int   `json:"points_restricted" gorm:"column:points_restricted;"`
	TimeTaken int64 `json:"totaltime_restricted" gorm:"column:totaltime_restricted;"`
}

type ContestTeam struct {
	Cid    int `json:"cid" gorm:"column:cid;"`
	TeamId int `json:"teamid" gorm:"column:teamid;"`
}

type ContestWelcomeEmail struct {
	ContestUrl       string `json:"contest_url"`
	Deadline         string `json:"deadline"`
	FirstName        string `json:"first_name"`
	Title            string `json:"title"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	ContestShortName string `json:"contest_short_name"`
	FromName         string `json:"from_name"`
}
