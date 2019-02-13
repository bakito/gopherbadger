package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/jpoles1/gopherbadger/coverbadge"
	"github.com/jpoles1/gopherbadger/logging"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

func getCommandOutput(commandString string) chan float64 {
	cmd := exec.Command("bash", "-c", commandString)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if nil != err {
		log.Fatalf("Error obtaining stdout: %s", err.Error())
	}
	reader := bufio.NewReader(stdout)
	coverageFloatChannel := make(chan float64)
	go func(reader io.Reader) {
		re := regexp.MustCompile("total:\\s*\\(statements\\)?\\s*(\\d+\\.?\\d*)\\s*\\%")
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			lineText := scanner.Text()
			match := re.FindStringSubmatch(lineText)
			if len(match) == 2 {
				color.Green(lineText)
				//fmt.Printf("Found coverage = %s%\n", match[1])
				coverageValue, err := strconv.ParseFloat(match[1], 32)
				errCheck("Parsing coverage to float", err)
				if err == nil {
					coverageFloatChannel <- coverageValue
				}
				break
			} else {
				fmt.Println(lineText)
			}
		}
		cmd.Wait()
	}(reader)
	if err := cmd.Start(); nil != err {
		log.Fatalf("Error starting program: %s, %s", cmd.Path, err.Error())
	}
	return coverageFloatChannel
}

func containsString(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func main() {
	badgeOutputFlag := flag.Bool("png", true, "Boolean to decide if .png will be generated by the software")
	badgeStyles := []string{"plastic", "flat", "flat-square", "for-the-badge", "social"}
	badgeStyleFlag := flag.String("style", "flat", "Badge style from list: ["+strings.Join(badgeStyles, ",")+"]")
	updateMdFilesFlag := flag.String("md", "", "A list of markdown filepaths for badge updates.")
	coveragePrefixFlag := flag.String("prefix", "Go", "A prefix to specify the coverage in your badge.")
	coverageCommandFlag := flag.String("covercmd", "", "gocover command to run; must print coverage report to stdout")
	manualCoverageFlag := flag.Float64("manualcov", -1.0, "A manually inputted coverage float.")
	tagsFlag := flag.String("tags", "", "The build tests you'd like to include in your coverage")
	flag.Parse()

	if !containsString(badgeStyles, *badgeStyleFlag) {
		logging.Fatal("Invalid style flag! Must be a member of list: ["+strings.Join(badgeStyles, ", ")+"]", errors.New("Invalid style flag"))
	}
	coverageBadge := coverbadge.Badge{
		CoveragePrefix: *coveragePrefixFlag,
		Style:          *badgeStyleFlag,
		ImageExtension: ".png",
	}
	var coverageFloat float64

	coverageCommand := ""
	if *coverageCommandFlag != "" {
		coverageCommand = *coverageCommandFlag
		if *tagsFlag != "" {
			log.Println("Warning: When the covercmd flag is used the tags flag will be ignored.")
		}
	} else if *tagsFlag != "" {
		coverageCommand = "go test ./... -tags \"" + *tagsFlag + "\" -coverprofile=coverage.out && go tool cover -func=coverage.out"
	} else {
		coverageCommand = "go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out"
	}

	if *manualCoverageFlag == -1 {
		coverageFloat = <-getCommandOutput(coverageCommand)
	} else {
		coverageFloat = *manualCoverageFlag
	}
	if *badgeOutputFlag == true {
		coverageBadge.DownloadBadge("coverage_badge.png", coverageFloat)
	}
	if *updateMdFilesFlag != "" {
		for _, filepath := range strings.Split(*updateMdFilesFlag, ",") {
			coverageBadge.WriteBadgeToMd(filepath, coverageFloat)
		}
	}
}
