package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

const notInUse = "not_in_use"

func main() {
	if checkRoot() {
		url := os.Getenv("CODIO_AUTOGRADE_V2_URL")
		os.Unsetenv("CODIO_AUTOGRADE_V2_URL")
		cleanup()
		createPaths()
		unzip("/autograder/source")
		prepareSubmission()
		execute()
		submitResults(url)
		// cleanup()
	} else {
		reExcuteRoot()
	}
}

func submitResults(urlPost string) {
	log.Println("Submit results from gradescope")
	jsonFile, err := os.Open("/autograder/results/results.json")
	check(err)
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var results gradescopeResult
	json.Unmarshal(byteValue, &results)

	log.Println("Submit results to Codio")
	score := fmt.Sprintf("%d", int64(results.Score))
	response, err := http.PostForm(urlPost,
		url.Values{"Grade": {score}, "Format": {"html"}})
	check(err)
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	check(err)
	log.Println("Done, response:")
	log.Println(body)
}

func prepareSubmission() {
	log.Println("Prepare submission")
	_, err := exec.Command("rsync", "-av", "--exclude", "autograder.zip", "/home/codio/workspace", "/autograder/submission").Output()
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

	_ = ioutil.WriteFile("/autograder/submission_metadata.json", file, 0644)
}

func execute() {
	log.Println("Copy run_autograde")
	_, err := copy("/autograder/source/run_autograder", "/autograder/run_autograder")
	check(err)
	os.Chmod("/autograder/run_autograder", 0777)

	log.Println("Executing run_autograde")
	autograde := exec.Command("/autograder/run_autograder")
	autograde.Dir = "/autograder/"
	stdoutFile, err := os.Create("/autograder/results/stdout")
	check(err)
	defer stdoutFile.Close()
	autograde.Stderr = stdoutFile
	autograde.Stdout = stdoutFile
	autograde.Start()
	autograde.Wait()
	log.Println(autograde.ProcessState.ExitCode())
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
	subProcess := exec.Command("sudo", "-E", ex)
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

func generateMetadata() {
	log.Println("generateMetadata")
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
	if !isRoot {
		return false
	}
	return true
}
