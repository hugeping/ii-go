# Intro

II-GO is [idec](https://ii-net.tk/) node realisation written in golang.

It has no dependencies and very compact. You can easy setup it and make your own ii/idec node.

How to build?

```
git clone https://github.com/gl00my/ii-go
cd ii-go
cd ii-tool
go build
cd ../ii-node
go build
```

# ii-tool

ii-tool can be used to fetch messages from another node and maintaince database.

## Fetch messages

ii-tool [options] fetch [uri] [echolist]

echolist is the file with echonames (can has : splitted columns, like list.txt) or - -- to load it from stdin. For example:

```
echo "std.club:this comment will be omitted" | ./ii-tool fecth http://127.0.0.1:8080 -
```

Options are:

```
-db \<database\> -- db by default (db.idx - genetated index)
-lim=<n>         -- fetch mode, if omitted full sync will be performed if needed
                    if n > 0 - last n messages synced
                    if n < 0 - adaptive fetching with step n will be performed
-f               -- do not check last message, perform sync even it is not needed
```

If echolist is omitted, fetcher will try to get all echos. It uses list.txt extension of IDEC if target node supports it.

## Create index

Index file (db.idx by default) is created when needed. If you want force to recreate it, useЖ

```
./ii-tool index
```

## Store bundle into db

DB is just msgid:message bundles in base64 stored in text file. You can merge records from db to db with store command:

```
ii-tool [options] store [db]
```
db - is file with bundles or '-' for stdin.

Options are:

```
-db \<database\> -- db to store/merge in;
```
## Show messages

Messages are identificated by unique message ids (MsgId). It is the first column in bundle:

```
<msgid>:<message>
```

You may select messages with select cmd:

```
./ii-tool [options] select <echo.name> [slice]
```

slice is the start:limit. For example:

```
./ii-tool select std.club -1:1 # get last message
./ii-tool select std.club 0:10 # get first 10 messages
```
Options are:

```
-db \<database\> -- db by default (db.idx - genetated index)
-v               -- show message text, not only MsgId
```

You may show selected message:

```
./ii-tool [options] get <MsgId>
```

Or search message:

```
./ii-tool [options] search <string> [echo]
```

Where options are:

```
-db \<database\> -- db by default (db.idx - genetated index)
-v               -- show message text, not only MsgId
```
To show messages adressed to selected user, try:

```
./ii-tool [options] c <user> [slice]
```
For example:

```
./ii-tool -v c Peter -1:1 # show and print last message
```

## Add user (point)

```
./ii-tool [-u pointfile] useradd <name> <e-mail> <password>
```

By default, pointfile is points.txt

## Blacklist msg

```
./ii-tool [-db db] blacklist <MsgId>
```
Blacklist is just new record with same id but spectial status.

# ii-node

To run node:

```
./ii-node [options]
```
Where options are:

```
-L               Listen address, default is :8080
-db <path>       Database, "db" by default
-e list          Echos list file. This file needs only for descriptions
                 and must be in list.txt format, where 2nd colum is ignored.
                 When this file is exists, points can not create they own echos.
                 list.txt by default.
-host <string>   Host string for node. For ex. http://hugeping.tk.
                 http://127.0.0.1:8080 by default
-sys "name"      Node name. "ii-go" by default
-u <points>      Points file. "points.txt" by default.
-v               Be verbose (for tracing)
```
## Example setup

```
cd ii-go/ii-node
ln -s ../ii-tool/ii-tool
./ii-tool fetch http://club.syscall.ru
wget http://club.syscall.ru/list.txt # for echo descriptions
./ii-node -sys "newnode"
```
And open http://127.0.0.1:8080 in your browser.

## Standarts supported

- u/e
- list.txt
- x/c
- list.txt
- blacklist.txt
- m/
- e/

## Limitations

Size of message can not be greater then 65536 bytes (before encoded into base64).

# web interface

User with id 1 (first created user) is admin.
Admin can create new echoes with: http://127.0.0.1:8080/new
Another hiden feature, is blacklisting: http://127.0.0.1:8080/msgid/blacklist

Web interface supports some non-standart features in message body text:

- @spolier: spoiler
- @base64: name (base64 data from next line till end of message)
- xpm2 and xpm3 images embedded

That's all, for now! :)
