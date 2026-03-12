
# Extract Log Files

Command Line Interface (CLI) application that functions to extract log data from source files to new files according to the date parameters to be extracted.


## Deployment

Build for linux:
```bash
  CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o appname -ldflags "-s -w"
```
Build for windows:
```bash
  CGO_ENABLED=0 GOOS="windows" GOARCH="amd64" go build -o appname.exe -ldflags "-s -w"
```


## Run Locally

Clone the project

```bash
  git clone https://github.com/herudins/ekstraklogfile.git
```

Go to the project directory

```bash
  cd my-project
```

Download dependencies:
```bash
  go mod tidy
```

Start the server

```bash
  go run -filename="path/filename.log" -start="2023-01-01 00:00" -end="2023-01-01 23:59" .
```


## Usage/Examples

```text
./appname -filename="path/filename.log" -start="2023-01-01 00:00" -end="2023-01-01 23:59"
```
## Supported date and time prefixes format
**Format**
* YYYY-MM-DD
* YYYY-MM-DD hh
* YYYY-MM-DD hh:mm
* YYYY-MM-DD hh:mm:ss

## Notes

Examples of supported log format prefixes:
```text
2023-01-01 01:01:01 [INFO] message log info
[2023-01-01 01:01:01] INFO message log info
```
Apart from that example, it is not yet supported.

## License

[MIT](https://choosealicense.com/licenses/mit/)
