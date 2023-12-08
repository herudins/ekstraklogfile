
# Ekstrak File Log

Aplikasi Command Line Interface (CLI) yang berfungsi untuk melakukan ekstraksi data log dari file sumber ke file baru sesuai dengan parameter tanggal yang ingin di ekstraksi.


## Deployment

Build aplikasi makefile linux dan windows:
```bash
  make build
```

Build aplikasi makefile linux:
```bash
  make build-linux
```

Build aplikasi makefile windows:
```bash
  make build-windows
```
Build aplikasi menggunakan compiler go:
```bash
  go build -o bin/nama_aplikasi
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
./nama_aplikasi -filename="path/filename.log" -start="2023-01-01 00:00" -end="2023-01-01 23:59"
```

## License

[MIT](https://choosealicense.com/licenses/mit/)


## Notes

Contoh prefix format log yang di support:
```text
2023-01-01 01:01:01 INFO message log info
[2023-01-01 01:01:01 INFO message log info]
```
Selain contoh tersebut belum di support.