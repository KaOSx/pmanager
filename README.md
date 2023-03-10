# pmanager
Packages viewer &amp; Mirrors status used by KaOS

## Installation

Required : Go ≥ 1.19, sqlite3

`go build -o pmanager pmanager.go`

## Configuration

Configuration is set at first launch of pmanager. It is located at /etc/pmanager/pmanager.conf.

Configuration variables :

* main section :
    - debug : (0|1) → Displays more output informations on debug and display not minified json
    - viewurl : URL of the frontend package viewer
    - repourl : Main repository
    - giturl : base URL of the github repository
    - logfile : file descriptor where to store the log (can be a file path, stdout (for standard output) or stderr (for standard error))
* database section :
    - uri : file path of the sqlite file
* repository section :
    - basedir : local folder where the packages repositories are stored
    - include : subfolders to include from analysis on update repositories databases action
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
    - mirrorlist : file where the list of the mirrors are set (can be a remote url or a locale file path)
    - pacmanconf : pacman configuration file (used to get the repos list – can be a remote url or a locale file path)

## Available subcommands

* update-repos : update the packages repositories database
* update-mirrors : update the mirrors status database
* update-all : update repos & mirrors
* serve : launch the webserver API (needed for the frontend)
* flag : launch an interactive prompt to manage the flagged packages
* test-mail : used to check the email configuration

All commands can be launched with the following options :

* --debug : force the debug mode whatever the configuration
* --no-debug : remove the debug mode whatever the configuration
* --log <filedescriptor> : override the log destination with the given file descriptor
