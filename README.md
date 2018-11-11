# pmanager
Packages viewer &amp; Mirrors status used by KaOS

## Installation

Required : Go ≥ 1.11

`go build -o pmanager pmanager.go`

## Configuration

Copy resources/etc/pmanager to /etc

Configuration variables :

* main section :
    - debug : (0|1) → Displays more output informations on debug and display not minified json
    - basedir : base directory for the database
    - viewurl : URL of the frontend package viewer
    - repourl : Main repository
    - giturl : base URL of the github repository
* database section :
    - subdir : subdir where the database files are stored
    - extension : suffix of the database files
* repository section :
    - basedir : local folder where the packages repositories are stored
    - exclude : subfolders to exclude from analysis on update repositories databases action
    - extension : suffix of the files where are stored packages informations of a repository
* api section :
    - port : port where the webserver api is launched
    - pagination : default number of results to return at a request
* smtp section :
    - host : smtp server
    - port : port of the smtp (usually 587 or 465)
    - use_encryption : (1|0) if 1, send email through STARTTLS
    - user : smtp user
    - password : smtp password
    - send_to : email address where the flag notifications are sent
    - send_from : email address used for the field “From:” of the notification emails
* mirror section :
    - main_mirror : base URL of the main mirror
    - mirrorlist : file where the list of the mirrors are set
    - pacmanconf : pacman configuration file (used to get the repos list)

## Available subcommands

* update-repos [<args…>] : update the packages repositories database. If args are used (name of a repo), just update the asked repos
* update-mirrors : update the mirrors status database
* update-all : update repos & mirrors
* serve : launch the webserver API (needed for the frontend)
* flag : launch an interactive prompt to manage the flagged packages
