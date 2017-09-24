# Depends Server / CLI

This project is meant to gather the data necessary for the "Depends" application. It connects to JIRA and pulls out the Threads, Requirements, Capablities, Features and Sprints. It uses the issues associated with sprints and the links in the previous items to build relationships. 

## To Run
Linux
```bash
export GOPATH=depends_svr
go get
go build
cd src/github.com/wtiger001/depends_svr
./depends_svr --user=<DI2E USER NAME> --password=<DI2E Password>
```

Windows
```bash
set GOPATH=depends_svr
go get
go build
cd src/github.com/wtiger001/depends_svr
./depends_svr.exe --user=<DI2E USER NAME> --password=<DI2E Password>
```

a file (output.json) will be created that can be imported into the depends tool
