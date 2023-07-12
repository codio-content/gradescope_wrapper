package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

const notInUse = "not_in_use"

func main() {
	args := os.Args[1:]
	runSetup := slices.Contains(args, "run-setup")
	extendedLogs := slices.Contains(args, "logs")

	if checkRoot() {
		url := os.Getenv("CODIO_AUTOGRADE_V2_URL")
		if len(url) == 0 {
			url = os.Getenv("CODIO_PARTIAL_POINTS_V2_URL")
		}
		if len(url) == 0 {
			panic("No Feedback URL, enable partial points.")
		}
		os.Unsetenv("CODIO_AUTOGRADE_V2_URL")
		os.Unsetenv("CODIO_PARTIAL_POINTS_V2_URL")
		os.Unsetenv("CODIO_AUTOGRADE_URL")
		os.Unsetenv("CODIO_PARTIAL_POINTS_URL")
		cleanup()
		createPaths()
		unzip("/autograder/source")
		if runSetup {
			executeSetupScript()
		}
		prepareSubmission()
		execute()
		submitResults(url, extendedLogs)
		cleanup()
	} else {
		reExcuteRoot()
	}
}

func executeSetupScript() {
	log.Println("run setup.sh")
	os.Chmod("/autograder/source/setup.sh", 0777)
	_, err := exec.Command("/autograder/source/setup.sh").Output()
	check(err)
}

func submitResults(urlPost string, extendedLogs bool) {
	log.Println("Read results from gradescope")
	exist, err := checkFileExists("/autograder/results/results.json")
	if err != nil || !exist {
		panic("Gradescope results file not found")
	}
	jsonFile, err := os.Open("/autograder/results/results.json")
	check(err)
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	check(err)

	var results gradescopeResult
	json.Unmarshal(byteValue, &results)
	log.Println("Submit results to Codio")
	score := fmt.Sprintf("%d", getScoreFromResult(results))
	urlValues := url.Values{"grade": {score}, "points": {score}, "feedback": {getFeedback(results, extendedLogs)}, "format": {"html"}}
	log.Println(urlValues)
	response, err := http.PostForm(urlPost, urlValues)

	check(err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	check(err)

	var codioOut codioResponse
	json.Unmarshal(body, &codioOut)
	log.Println("Done, response:")
	if codioOut.Code != 1 {
		panic(fmt.Sprintf("Response %d: %s", codioOut.Code, codioOut.Message))
	}
}

func getScoreFromResult(results gradescopeResult) int64 {
	if results.Score != nil {
		//overriden score
		return int64(math.Ceil(*results.Score))
	}
	var totalScore float64
	var assignedScore float64
	totalScore = 0
	assignedScore = 0
	for _, test := range results.Tests {
		totalScore += test.MaxScore
		assignedScore += test.Score
	}
	return int64(math.Ceil(assignedScore / totalScore * 100))
}

func getFeedback(results gradescopeResult, extendedLogs bool) string {
	var output strings.Builder
	output.WriteString("<p>")
	output.WriteString("Total Points<br/>")
	score := fmt.Sprintf("<b>%d / 100</b><br/>", getScoreFromResult(results))
	output.WriteString(score)
	failedTests := filterTests(results.Tests, false)
	passedTests := filterTests(results.Tests, true)
	if len(failedTests) > 0 {
		output.WriteString("<br/><p style='color: #b94a48'>")
		output.WriteString("<b>Failed Tests</b><br/>")
		for _, test := range failedTests {
			if extendedLogs {
				printFeedbackTestAdvanced(&output, test)
			} else {
				printFeedbackTest(&output, test)
			}
		}
		output.WriteString("</p>")
	}

	if len(passedTests) > 0 {
		output.WriteString("<br/><p style='color: #468847'>")
		output.WriteString("<b>Passed Tests</b><br/>")
		for _, test := range passedTests {
			if extendedLogs {
				printFeedbackTestAdvanced(&output, test)
			} else {
				printFeedbackTest(&output, test)
			}
		}
		output.WriteString("</p>")
	}
	output.WriteString("</p>")
	return output.String()
}

func printFeedbackTest(output *strings.Builder, test gradescopeResultTests) {
	output.WriteString(fmt.Sprintf("%s (%.2f/%.2f)<br/>", test.Name, test.Score, test.MaxScore))
}

func printFeedbackTestAdvanced(output *strings.Builder, test gradescopeResultTests) {
	output.WriteString("<details><summary>")
	output.WriteString(fmt.Sprintf("%s (%.2f/%.2f)<br/>", test.Name, test.Score, test.MaxScore))
	output.WriteString("</summary><pre>")
	output.WriteString(test.Output)
	output.WriteString("</pre></details>")
}

func filterTests(tests []gradescopeResultTests, flag bool) []gradescopeResultTests {
	var res []gradescopeResultTests
	for _, test := range tests {
		if (flag && (test.Status == "passed" || test.Score >= test.MaxScore)) ||
			(!flag && (test.Status == "failed" || test.Score < test.MaxScore)) {
			res = append(res, test)
		}
	}
	return res
}

func prepareSubmission() {
	log.Println("Prepare submission")
	_, err := exec.Command("rsync", "-av", "--exclude", "autograder.zip", "--exclude", "gradescope_wrapper",
		"--exclude", ".guides", "--exclude", ".codio", "--exclude", ".settings",
		"/home/codio/workspace/", "/autograder/submission").Output()
	check(err)
	log.Println("Prepare submission info")

	var codioSubmissionInfo codioAutograde
	err = json.Unmarshal([]byte(os.Getenv("CODIO_AUTOGRADE_ENV")), &codioSubmissionInfo)
	check(err)

	submissionInfo := gradescopeSubmission{
		Id:        notInUse,
		CreatedAt: codioSubmissionInfo.CompletedDate,
		Assignment: gradescopeSubmissionAssignment{
			Id:          codioSubmissionInfo.Course.Assignment.Id,
			Title:       notInUse,
			CourseId:    codioSubmissionInfo.Course.Id,
			ReleaseDate: codioSubmissionInfo.Course.Assignment.Start,
			LateDueDate: codioSubmissionInfo.Course.Assignment.End,
			TotalPoints: "100.0",
		},
		SubmissionMethod: "upload",
		Users: []gradescopeSubmissionUser{
			{
				Email: codioSubmissionInfo.Student.Email,
				Id:    codioSubmissionInfo.Student.Id,
				Name:  codioSubmissionInfo.Student.FullName,
			},
		},
		PreviousSubmissions: []gradescopeSubmissionPrevious{},
	}

	file, _ := json.MarshalIndent(submissionInfo, "", " ")

	_ = os.WriteFile("/autograder/submission_metadata.json", file, 0644)
}

func execute() {
	log.Println("Copy run_autograde")
	_, err := copy("/autograder/source/run_autograder", "/autograder/run_autograder")
	check(err)
	os.Chmod("/autograder/run_autograder", 0777)

	log.Println("Executing run_autograde")
	path := os.Getenv("PATH")
	autograde := exec.Command("sudo", "-E", "-u", "codio", "env", fmt.Sprintf("PATH=%s", path), "/autograder/run_autograder")
	autograde.Dir = "/autograder/"
	stdoutFile, err := os.Create("/autograder/results/stdout")
	check(err)
	_, err = exec.Command("chown", "codio:codio", "-R", "/autograder/").Output()
	check(err)
	defer stdoutFile.Close()
	autograde.Stderr = stdoutFile
	autograde.Stdout = stdoutFile
	autograde.Start()
	autograde.Wait()
	log.Printf("Exite Code: %d\n", autograde.ProcessState.ExitCode())
	// uncomment to debug output
	// stdpoutFile, _ := os.ReadFile("/autograder/results/stdout")
	// fmt.Printf("OUTPUT:\n%s\n", stdpoutFile)
	if autograde.ProcessState.ExitCode() != 0 {
		panic(fmt.Sprintf("run_autograde failed with %d", autograde.ProcessState.ExitCode()))
	}
}

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func reExcuteRoot() {
	log.Println("switching to root")
	ex, err := os.Executable()
	check(err)
	path := os.Getenv("PATH")
	args := []string{"-E", "env", fmt.Sprintf("PATH=%s", path), ex}
	args = append(args, os.Args[1:]...)
	subProcess := exec.Command("sudo", args...)
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr
	subProcess.Start()
	subProcess.Wait()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func checkFileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func unzip(destination string) error {
	autograderFile := "/home/codio/workspace/.guides/autograder.zip"
	exist, err := checkFileExists(autograderFile)
	check(err)
	if !exist {
		autograderFile = "/home/codio/workspace/.guides/secure/autograder.zip"
		exist, err = checkFileExists(autograderFile)
		check(err)
		if !exist {
			panic("autograder.zip not found in .guides/ or .guides/secure")
		}
	}

	archive, err := zip.OpenReader(autograderFile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, file := range archive.Reader.File {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()
		path := filepath.Join(destination, file.Name)
		// Remove file if it already exists; no problem if it doesn't; other cases can error out below
		_ = os.Remove(path)
		// Create a directory at path, including parents
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
		// If file is _supposed_ to be a directory, we're done
		if file.FileInfo().IsDir() {
			continue
		}
		// otherwise, remove that directory (_not_ including parents)
		err = os.Remove(path)
		if err != nil {
			return err
		}
		// and create the actual file.  This ensures that the parent directories exist!
		// An archive may have a single file with a nested path, rather than a file for each parent dir
		writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer writer.Close()
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	}
	return nil
}

func createPaths() {
	log.Println("createPaths")
	err := os.Mkdir("/autograder", 0755)
	check(err)

	err = os.Mkdir("/autograder/source", 0755)
	check(err)

	err = os.Mkdir("/autograder/results", 0755)
	check(err)
	err = os.Mkdir("/autograder/submission", 0755)
	check(err)

}

func cleanup() {
	log.Println("cleanup")
	err := os.RemoveAll("/autograder")
	if err != nil {
		log.Println(err)
	}
}

func checkRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("[isRoot] Unable to get current user: %s", err)
		panic(err)
	}
	isRoot := currentUser.Username == "root"
	return isRoot
}
