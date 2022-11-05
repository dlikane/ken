package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	offset = 0
	Execute()
}

var (
	rootCmd = &cobra.Command{
		Use:   "srt",
		Short: "Convert org srt_file.txt file to srt_file_ru.srt and srt_file_eu.srt",
		Long:  `Quick and dirty one`,
		Run: func(cmd *cobra.Command, args []string) {
			processFile(inputFilename)
		},
	}
	inputFilename string
	offset        int
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&inputFilename, "inputFile", "i", "./srt.txt", "input file (default is ./srt.txt)")
}

func Execute() error {
	return rootCmd.Execute()
}

type subtitle struct {
	btime   string
	etime   string
	russian string
	english string
	offset  int
}

func parseTime(s string, ln int) (bt string, et string) {
	arr := strings.Split(s, " ")
	for i, ss := range arr {
		if len(ss) != 4 && len(ss) != 5 {
			logrus.Fatalf("line %d expect timestamp in format 00:00 [00:00], got %s and %s", ln, s, ss)
		}
		if len(ss) == 4 {
			ss = "0" + ss
		}
		if ss[2] != ':' {
			logrus.Fatalf("line %d expect timestamp in format 00:00 [00:00], got %s and %s", ln, s, ss)
		}
		switch i {
		case 0:
			bt = ss
		default:
			et = ss
		}
	}
	return bt, et
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := strings.TrimRight(scanner.Text(), " \n\r\t")

		lines = append(lines, str)
	}
	return lines, scanner.Err()
}

func readSubtitles(path string) []subtitle {
	var subtitels []subtitle
	lines, err := readLines(path)
	if err != nil {
		logrus.WithError(err).Error("can't load file")
	}
	lnCnt := 1
	lnCntTotal := 1
	var st *subtitle
	for _, str := range lines {
		if strings.HasPrefix(str, "+") {
			o, err := strconv.ParseUint(str[1:], 10, 32)
			if err != nil {
				logrus.Fatalf("line: %d invalid offset format: %s", lnCntTotal, str)
			}
			offset += int(o)
			continue
		}
		switch lnCnt {
		case 1:
			if str == "" {
				lnCnt = 0
			} else {
				st = &subtitle{}
				st.btime, st.etime = parseTime(str, lnCntTotal)
				st.offset = offset
			}
		case 2:
			st.russian = str
		case 3:
			st.english = str
		default:
			if str != "" {
				logrus.Fatalf("line: %d expect empty line, got: %s", lnCntTotal, str)
			}
			if st != nil {
				subtitels = append(subtitels, *st)
			}
			lnCnt = 0
		}
		lnCnt++
		lnCntTotal++
	}
	if st != nil {
		subtitels = append(subtitels, *st)
	}
	return subtitels
}

func dualWrite(w1 *bufio.Writer, w2 *bufio.Writer, s string) {
	_, _ = w1.WriteString(s + "\n")
	_, _ = w2.WriteString(s + "\n")
}

func formatTime(st1 string, st2 string, offset int) string {
	if st1 > st2 {
		logrus.Fatalf("oops - revers time: %s %s", st1, st2)
	}

	return fmt.Sprintf("00:%s,000 --> 00:%s,000", timeWithOffset(st1, offset), timeWithOffset(st2, offset))
}

func timeWithOffset(s string, o int) string {
	min, _ := strconv.ParseUint(s[0:2], 10, 32)
	sec, _ := strconv.ParseUint(s[3:], 10, 32)
	tt := int(min)*60 + int(sec) + o
	return fmt.Sprintf("%02d:%02d", tt/60, tt%60)
}

func writeSubtitles(basefilename string, subtitles []subtitle) {
	fileRu, err := os.OpenFile(basefilename+"_ru.srt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.WithError(err).Fatalf("can't write file")
	}
	defer fileRu.Close()
	fileEn, err := os.OpenFile(basefilename+"_en.srt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.WithError(err).Fatalf("can't write file")
	}
	defer fileEn.Close()
	wRu := bufio.NewWriter(fileRu)
	defer wRu.Flush()
	wEn := bufio.NewWriter(fileEn)
	defer wEn.Flush()
	cnt := 1
	for i := range subtitles {
		if i == len(subtitles)-1 {
			break
		}
		st := subtitles[i]
		et := st.etime
		if et == "" {
			et = subtitles[i+1].btime
		}
		ts := formatTime(st.btime, et, st.offset)
		dualWrite(wRu, wEn, fmt.Sprintf("%d", cnt))
		dualWrite(wRu, wEn, ts)
		_, _ = wRu.WriteString(st.russian + "\n")
		_, _ = wEn.WriteString(st.english + "\n")
		dualWrite(wRu, wEn, "")
		cnt++
	}
}

func processFile(filename string) {
	wd, _ := os.Getwd()
	logrus.Infof("Processing file: %s in %s", filename, wd)
	ext := filepath.Ext(filename)
	subtitles := readSubtitles(filename)
	basefilename := filename[:len(filename)-len(ext)]
	writeSubtitles(basefilename, subtitles)
	logrus.Infof("Done: %d subtitles", len(subtitles))
}
