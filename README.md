# TUIpe Racer

TUI based typing test written in Go

Compete against yourself to improve your typing speed with this terminal typing test!

<h1>Installing</h1>
<h3>Required</h3>

```
golang
gcc
```

clone repo with
```
git clone https://github.com/ethanamaher/tuiperacer.git
```
and cd into the tuiperacer directory
```
go mod tidy
go build -x
```
the build command can take awhile so the -x flag is added to see the commands that are run

you may also need to do
```
go env -w CGO_ENABLED=1
```
if it says something about it

from there you can run the file
```
./tuiperacer <word_count>
```
with default wordcount of 15 if not specified

![image](https://github.com/user-attachments/assets/f663a4ce-de96-40b6-b73c-9e51e02efc09)

