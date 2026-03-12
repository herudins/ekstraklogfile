package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	dateTimeSecond = "2006-01-02 15:04:05"

	// buffer size
	ReadBufferSize  = 512 * 1024
	WriteBufferSize = 512 * 1024

	// Konstanta multiplier untuk membuat timestamp key
	yearMultiplier  int64 = 10000000000
	monthMultiplier int64 = 100000000
	dayMultiplier   int64 = 1000000
	hourMultiplier  int64 = 10000
	minMultiplier   int64 = 100
)

// Kebutuhan keterangan ke user
var (
	thnBlnHari              = "tahun-bulan-hari"
	thnBlnHariJam           = thnBlnHari + " jam"
	thnBlnHariJamMenit      = thnBlnHariJam + ":menit"
	thnBlnHariJamMenitDetik = thnBlnHariJamMenit + ":detik"
	strDate                 = thnBlnHari + "\n" + thnBlnHariJam + "\n" + thnBlnHariJamMenit + "\n" + thnBlnHariJamMenitDetik
	descStartDate           = "Filter tanggal awal, format yang saat ini di support:\n" + strDate
	descEndDate             = "Filter tanggal akhir, format harus identik dengan tanggal awal. \nFormat yang saat ini di support:\n" + strDate
)

// Untuk menyimpan batas waktu filter
type TimestampRange struct {
	Start int64
	End   int64
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	//Buat logger untuk error proses
	appLog, err := os.OpenFile("application.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Error create new file:", err.Error())
		return
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
		return
	}
	if *startDate == "" {
		logger.Error("parameter start wajib diisi.")
		return
	}
	if *endDate == "" {
		logger.Error("parameter end wajib diisi.")
		return
	}

	var (
		filePath   = *filename
		startInput = *startDate
		endInput   = *endDate
	)

	// Panjang waktu awal dan akhir harus sama
	if len(startInput) != len(endInput) {
		logger.Error("start and end time format must be the same")
		return
	}

	// Validasi format dan nilai waktu awal
	if !validateUserTime(startInput) {
		logger.Error("invalid start time format/value")
		return
	}

	// Validasi format dan nilai waktu akhir
	if !validateUserTime(endInput) {
		logger.Error("invalid end time format/value")
		return
	}

	// Validasi range waktu awal dan akhir
	timeRange := buildTimestampRange(startInput, endInput)
	if timeRange.Start > timeRange.End {
		logger.Error("start time must be before end time")
		return
	}

	//Open file log yang akan di filter
	inputFile, err := os.Open(filePath)
	if err != nil {
		logger.Error("cannot open file: " + err.Error())
		return
	}
	defer inputFile.Close()

	//Buat output file log hasil filter
	outputFileName := fmt.Sprintf("log_ekstrak_%s_%s.log", sanitizeFilenamePart(startInput), sanitizeFilenamePart(endInput))
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("cannot create output file:", err)
		return
	}
	defer outputFile.Close()

	//Siapkan buffer untuk reader dan writer
	reader := bufio.NewReaderSize(inputFile, ReadBufferSize)
	writer := bufio.NewWriterSize(outputFile, WriteBufferSize)

	//Waktu pertama baca file log
	start := time.Now()
	startLog := start.Format(dateTimeSecond)

	//Lakukan filter log
	readCount, writeCount, err := filterLog(reader, writer, timeRange)
	if err != nil {
		logger.Error("error when read file: " + err.Error())
	}
	writer.Flush()

	//Hitung berapa lama waktu yang dibutuhkan
	processDuration := time.Since(start)
	end := start.Add(processDuration)
	endLog := end.Format(dateTimeSecond)

	unreadCount := max(countUnreadData(reader), 0)
	totalLine := readCount + unreadCount

	msgLog := fmt.Sprintf("Sukses ekstrak file log ke file baru dengan nama: %s. Jumlah hasil ekstrak adalah %d baris dari baris yang dibaca sebanyak: %d dari total %d baris.", outputFileName, writeCount, readCount, totalLine)
	logger.Info(msgLog)

	logTime := fmt.Sprintf("Proses baca file berlangsung selama %.2f detik dari %s s/d %s", processDuration.Seconds(), startLog, endLog)
	logger.Info(logTime)
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
	b.WriteString("    [2023-01-01 01:01:01] message log info\n")
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

// filterLog membaca log baris per baris dan menulis hanya log
// yang berada dalam rentang waktu tertentu
func filterLog(reader *bufio.Reader, writer *bufio.Writer, timeRange TimestampRange) (int64, int64, error) {
	var shouldWrite bool
	var readCount int64
	var writeCount int64
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return readCount, writeCount, err
			}

			if len(line) > 0 {
				if shouldWrite {
					writer.Write(line)
					writeCount++
				}
			}
			break
		}
		readCount++

		// Periksa barisnya apakah memiliki prefix timestamp atau tidak
		// kalau timestamp maka bandingkan dengan range waktu dari user
		if ts, ok := extractLogAndCheckTimestamp(line); ok {
			if ts >= timeRange.Start && ts <= timeRange.End {
				shouldWrite = true
			} else {
				shouldWrite = false
				break
			}
		}

		//Disini barisnya bukan timestamp, maka cek apakah masih harus menulis log atau tidak
		if shouldWrite {
			writer.Write(line)
			writeCount++
		}
	}

	return readCount, writeCount, nil
}

func countUnreadData(reader *bufio.Reader) int64 {
	var count int64
	for {
		_, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				count = -1
				break
			}

			return count
		}
		count++
	}

	return count
}

// extractLogAndCheckTimestamp mencoba membaca timestamp dari prefix log
func extractLogAndCheckTimestamp(line []byte) (int64, bool) {
	//Jika kurang dari 19 digit maka itu fix bukan prefix timestamp
	if len(line) < 19 {
		return 0, false
	}

	prefix := line[:19]

	// Format: prefix memiliki karakter [
	if line[0] == '[' {
		if len(line) < 20 {
			return 0, false
		}

		prefix = line[1:20]
	}

	if !validateTimeFormat(prefix) {
		return 0, false
	}

	return parseTimestampBytes(prefix), true
}

// parseTimestampBytes mengubah timestamp text menjadi numeric key
func parseTimestampBytes(b []byte) int64 {
	year := parse4DigitNumber(b, 0)
	month := parse2DigitNumber(b, 5)
	day := parse2DigitNumber(b, 8)
	hour := parse2DigitNumber(b, 11)
	minute := parse2DigitNumber(b, 14)
	second := parse2DigitNumber(b, 17)

	return makeTimestampKey(year, month, day, hour, minute, second)
}

// buildTimestampRange membuat batas timestamp numeric
func buildTimestampRange(startInput, endInput string) TimestampRange {
	startMin, _ := expandUserTimeRange(startInput)
	_, endMax := expandUserTimeRange(endInput)

	return TimestampRange{
		Start: startMin,
		End:   endMax,
	}
}

// expandUserTimeRange memperluas input user menjadi range minimum dan maksimum
func expandUserTimeRange(input string) (int64, int64) {
	bytesInput := []byte(input)
	year := parse4DigitNumber(bytesInput, 0)
	month := parse2DigitNumber(bytesInput, 5)
	day := parse2DigitNumber(bytesInput, 8)

	hour := 0
	minute := 0
	second := 0

	length := len(bytesInput)

	if length >= 13 {
		hour = parse2DigitNumber(bytesInput, 11)
	}

	if length >= 16 {
		minute = parse2DigitNumber(bytesInput, 14)
	}

	if length >= 19 {
		second = parse2DigitNumber(bytesInput, 17)
	}

	min := makeTimestampKey(year, month, day, hour, minute, second)

	switch length {
	case 10:
		return min, makeTimestampKey(year, month, day, 23, 59, 59)

	case 13:
		return min, makeTimestampKey(year, month, day, hour, 59, 59)

	case 16:
		return min, makeTimestampKey(year, month, day, hour, minute, 59)
	}

	return min, min
}

// makeTimestampKey membuat integer timestamp agar mudah dibandingkan
func makeTimestampKey(year, month, day, hour, minute, second int) int64 {
	return int64(year)*yearMultiplier +
		int64(month)*monthMultiplier +
		int64(day)*dayMultiplier +
		int64(hour)*hourMultiplier +
		int64(minute)*minMultiplier +
		int64(second)
}

// Fungsi untuk kebutuhan parsing 2 digit number
func parse2DigitNumber(b []byte, i int) int {
	return int(b[i]-'0')*10 + int(b[i+1]-'0')
}

// Fungsi untuk kebutuhan parsing 4 digit number
func parse4DigitNumber(b []byte, i int) int {
	return int(b[i]-'0')*1000 +
		int(b[i+1]-'0')*100 +
		int(b[i+2]-'0')*10 +
		int(b[i+3]-'0')
}

// validateUserTime memvalidasi format dan nilai waktu user
func validateUserTime(input string) bool {
	if !validateTimeFormat([]byte(input)) {
		return false
	}

	bytesData := []byte(input)

	year := parse4DigitNumber(bytesData, 0)
	month := parse2DigitNumber(bytesData, 5)
	day := parse2DigitNumber(bytesData, 8)

	if month < 1 || month > 12 {
		return false
	}

	if day < 1 || day > daysInMonth(year, month) {
		return false
	}

	switch len(bytesData) {

	case 10:
		return true

	case 13:
		hour := parse2DigitNumber(bytesData, 11)
		return hour >= 0 && hour <= 23

	case 16:
		hour := parse2DigitNumber(bytesData, 11)
		minute := parse2DigitNumber(bytesData, 14)
		return hour <= 23 && minute <= 59

	case 19:
		hour := parse2DigitNumber(bytesData, 11)
		minute := parse2DigitNumber(bytesData, 14)
		second := parse2DigitNumber(bytesData, 17)
		return hour <= 23 && minute <= 59 && second <= 59
	}

	return false
}

// validateTimeFormat memvalidasi struktur format waktu
// 10 digit => YYYY-MM-DD
// 13 digit => YYYY-MM-DD hh
// 16 digit => YYYY-MM-DD hh:mm
// 19 digit => YYYY-MM-DD hh:mm:ss
func validateTimeFormat(input []byte) bool {
	switch len(input) {
	case 10:
		return validateDate(input)

	case 13:
		return validateDate(input) &&
			input[10] == ' ' &&
			isDigit(input[11]) &&
			isDigit(input[12])

	case 16:
		return validateDate(input) &&
			input[10] == ' ' &&
			isDigit(input[11]) &&
			isDigit(input[12]) &&
			input[13] == ':' &&
			isDigit(input[14]) &&
			isDigit(input[15])

	case 19:
		return validateDate(input) &&
			input[10] == ' ' &&
			isDigit(input[11]) &&
			isDigit(input[12]) &&
			input[13] == ':' &&
			isDigit(input[14]) &&
			isDigit(input[15]) &&
			input[16] == ':' &&
			isDigit(input[17]) &&
			isDigit(input[18])
	}

	return false
}

// validateDate memastikan format YYYY-MM-DD valid
func validateDate(b []byte) bool {

	if len(b) < 10 {
		return false
	}

	return isDigit(b[0]) &&
		isDigit(b[1]) &&
		isDigit(b[2]) &&
		isDigit(b[3]) &&
		b[4] == '-' &&
		isDigit(b[5]) &&
		isDigit(b[6]) &&
		b[7] == '-' &&
		isDigit(b[8]) &&
		isDigit(b[9])
}

// isDigit mengecek apakah karakter adalah angka
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// daysInMonth mengembalikan jumlah hari dalam bulan
func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	}

	return 0
}

// isLeapYear menentukan apakah tahun adalah tahun kabisat atau bukan
func isLeapYear(year int) bool {
	if year%400 == 0 {
		return true
	}
	if year%100 == 0 {
		return false
	}

	return year%4 == 0
}

// sanitizeFilenamePart menghapus karakter -, spasi dan :
// agar cocok dijadikan nama file
//
// contoh:
// 2026-01-01 -> 20260101
// 2026-01-01 10:30 -> 202601011030
func sanitizeFilenamePart(input string) string {
	resByte := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		char := input[i]
		if char == '-' || char == ' ' || char == ':' {
			continue
		}
		resByte = append(resByte, char)
	}

	return string(resByte)
}
