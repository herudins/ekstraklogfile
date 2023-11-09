package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	date           = "2006-01-02"
	dateTimeHour   = "2006-01-02 15"
	dateTimeMinute = "2006-01-02 15:04"
	dateTimeSecond = "2006-01-02 15:04:05"
	javaPrefix     = "["
)

var (
	thnBlnHari              = "tahun-bulan-hari"
	thnBlnHariJam           = thnBlnHari + " jam"
	thnBlnHariJamMenit      = thnBlnHariJam + ":menit"
	thnBlnHariJamMenitDetik = thnBlnHariJamMenit + ":detik"
	strDate                 = thnBlnHari + "\n" + thnBlnHariJam + "\n" + thnBlnHariJamMenit + "\n" + thnBlnHariJamMenitDetik
	descStartDate           = "Filter tanggal awal, format yang saat ini di support:\n" + strDate
	descEndDate             = "Filter tanggal akhir, format harus identik dengan tanggal awal. \nFormat yang saat ini di support:\n" + strDate
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	//Buat logger untuk error proses
	appLog, err := os.OpenFile("application.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Error create new file:", err.Error())
		os.Exit(1)
	}
	defer func() {
		_ = appLog.Close()
	}()

	//Konfigurasi logger
	loggerWriter := io.MultiWriter(os.Stdout, appLog)
	logger := configureLogger(loggerWriter)

	//Parsing flag
	var (
		filename  = flag.String("filename", "", "Nama file log yang akan di ekstrak, full path.")
		startDate = flag.String("start", "", descStartDate)
		endDate   = flag.String("end", "", descEndDate)
	)

	flag.Usage = usageFlag
	flag.Parse()

	if *filename == "" {
		logger.Error("parameter filename wajib diisi.")
		os.Exit(1)
	}
	if *startDate == "" {
		logger.Error("parameter start wajib diisi.")
		os.Exit(1)
	}
	if *endDate == "" {
		logger.Error("parameter end wajib diisi.")
		os.Exit(1)
	}

	var (
		filePath       = *filename
		prefixArgStart = *startDate
		prefixArgEnd   = *endDate
	)

	//Cek format prefix tanggal awal
	dtStart, formatDt, err := cekFormatPrefix(prefixArgStart)
	if err != nil {
		logger.Error(err.Error() + " datetime awal: " + prefixArgStart)
		os.Exit(1)
	}

	//Parse prefix tanggal akhir, karena harus sama dengan tanggal awal
	dtEnd, err := time.Parse(formatDt, prefixArgEnd)
	if err != nil {
		logger.Error("format datetime akhir tidak sama dengan format datetime awal")
		os.Exit(1)
	}

	//Cek tanggal akhir tidak boleh kurang dari tanggal awal
	if dtEnd.Before(dtStart) {
		logger.Error("datetime akhir harus besar dari datetime awal")
		os.Exit(1)
	}

	//Baca file log
	filePath = filepath.FromSlash(filePath)
	readFile, err := os.Open(filePath)
	if err != nil {
		logger.Error("Error open file: " + err.Error())
		os.Exit(1)
	}
	defer func() {
		_ = readFile.Close()
	}()

	// Buat nama file untuk output
	var (
		replacer        = strings.NewReplacer("-", "", " ", "", ":", "")
		prefixFileStart = replacer.Replace(prefixArgStart)
		prefixFileEnd   = replacer.Replace(prefixArgEnd)
		newFileName     = fmt.Sprintf("log_ekstrak_%s_%s.log", prefixFileStart, prefixFileEnd)
	)

	//Buat file log baru untuk menampung hasil filter log
	file, err := os.OpenFile(newFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logger.Error("Error create new file: " + err.Error())
		os.Exit(1)
	}
	defer func() {
		_ = file.Close()
	}()

	//Create writer
	w := bufio.NewWriter(file)
	defer func() {
		_ = w.Flush()
	}()

	var (
		isWriting     = false
		lPrefArgStart = len(prefixArgStart)
	)

	//Untuk log berapa lama baca file berlangsung
	start := time.Now()
	startLog := start.Format(dateTimeSecond)
	defer func() {
		processDuration := time.Since(start)
		end := start.Add(processDuration)
		endLog := end.Format(dateTimeSecond)

		logTime := fmt.Sprintf("Proses baca file berlangsung selama %.2f detik dari %s s/d %s", processDuration.Seconds(), startLog, endLog)
		logger.Info(logTime)
	}()

	//Create reader
	noRead := 0
	noWrite := 0
	reader := bufio.NewReader(readFile)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				logger.Info("Selesai membaca file.")
			} else {
				logger.Error("error readline: " + err.Error())
			}

			break
		}
		text := string(line)
		noRead++

		//Cek panjang text apakah lebih panjang dari prefix atau tidak
		//Jika kurang dari panjang prefix dan sedang dalam mode menulis, maka lanjutkan menulis ke log baru
		if len(text) < lPrefArgStart {
			if isWriting {
				_, err := w.WriteString(text + "\n")
				if err != nil {
					logger.Error("Error write text to file: " + err.Error())
					break
				}
				noWrite++

				continue
			}
		}

		//Ambil prefix text, untuk nanti di cek
		textPrefix := strings.TrimSpace(text)
		if len(textPrefix) > lPrefArgStart {
			textPrefix = strings.TrimLeft(textPrefix, javaPrefix)
			textPrefix = textPrefix[0:lPrefArgStart]
		}

		//Parse text prefix ke datetime sesuai argumen
		//Jika error parse tetapi masih dalam mode menulis, maka simpan teks ke log
		dtData, err := time.Parse(formatDt, textPrefix)
		if err != nil {
			if isWriting {
				_, err := w.WriteString(text + "\n")
				if err != nil {
					logger.Error("Error write text to file: " + err.Error())
					break
				}
				noWrite++
			}

			continue
		}

		//Cek jika datetime nya sama dengan awal atau setelah awal dan datetimenya sama dengan akhir atau sebelum akhir maka tulis ke log baru
		//Jika lebih dari akhir maka berhenti saja menulis dan baca log nya
		if (dtData.Equal(dtStart) || dtData.After(dtStart)) && (dtData.Equal(dtEnd) || dtData.Before(dtEnd)) {
			_, err := w.WriteString(text + "\n")
			if err != nil {
				logger.Error("Error write text to file:" + err.Error())
				break
			}
			isWriting = true
			noWrite++
		} else if dtData.After(dtEnd) {
			logger.Info("Proses baca file log selesai")

			isWriting = false
			break
		}
	}

	totalLine, err := countLines(filePath)
	if err != nil {
		logger.Info("Error ambil informasi jumlah baris:" + err.Error())
		totalLine = -1
	}

	msgLog := fmt.Sprintf("Sukses ekstrak file log ke file baru dengan nama: %s. Jumlah hasil ekstrak adalah %d baris dari baris yang dibaca sebanyak: %d dari total %d baris.", newFileName, noWrite, noRead, totalLine)
	logger.Info(msgLog)
}

func usageFlag() {
	flagSet := flag.CommandLine
	order := []string{"filename", "start", "end"}
	var b strings.Builder
	for _, name := range order {
		flag := flagSet.Lookup(name)
		b.WriteString("  -" + flag.Name)
		b.WriteString("\n    \t")

		usage := strings.ReplaceAll(flag.Usage, "\n", "\n    \t")
		b.WriteString(usage + "\n")
	}
	b.WriteString("\nContoh penggunaan:\n")
	b.WriteString("    ./nama_aplikasi -filename=\"path/filename.log\" -start=\"2023-01-01 00:00\" -end=\"2023-01-01 23:59\"\n")

	b.WriteString("\nContoh prefix format log yang di support:\n")
	b.WriteString("    2023-01-01 01:01:01 INFO message log info\n")
	b.WriteString("    [2023-01-01 01:01:01 INFO message log info]\n")
	b.WriteString("Selain contoh tersebut belum di support.\n")

	str := b.String()
	b.Reset()

	fmt.Println(str)
}

func configureLogger(writer io.Writer) *slog.Logger {
	logOpt := slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				timeV := a.Value.Time()
				timeS := timeV.Format(dateTimeSecond)
				a.Value = slog.StringValue(timeS)
			}

			return a
		},
	}

	return slog.New(slog.NewJSONHandler(writer, &logOpt))
}

func cekFormatPrefix(prefix string) (time.Time, string, error) {
	dt, err := time.Parse(date, prefix)
	if err == nil {
		return dt, date, nil
	}

	dt, err = time.Parse(dateTimeHour, prefix)
	if err == nil {
		return dt, dateTimeHour, nil
	}

	dt, err = time.Parse(dateTimeMinute, prefix)
	if err == nil {
		return dt, dateTimeMinute, nil
	}

	dt, err = time.Parse(dateTimeSecond, prefix)
	if err == nil {
		return dt, dateTimeSecond, nil
	}

	return dt, "", errors.New("invalid datetime prefix format")
}

func countLines(path string) (int, error) {
	readFile, err := os.Open(path)
	if err != nil {
		return -1, err
	}
	defer func() {
		_ = readFile.Close()
	}()

	var count int
	var read int
	var target []byte = []byte("\n")

	buffer := make([]byte, 32*1024)
	r := bufio.NewReader(readFile)
	for {
		read, err = r.Read(buffer)
		if err != nil {
			break
		}

		count += bytes.Count(buffer[:read], target)
	}

	if err == io.EOF {
		return count, nil
	}

	return count, err
}
