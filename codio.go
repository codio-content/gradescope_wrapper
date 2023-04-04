package main

type codioAutograde struct {
	// Assessments []codioAutogradeAssessments `json:"assessments"` //TODO ?
	CompletedDate string                `json:"completedDate"`
	Student       codioAutogradeStudent `json:"student"`
	Course        codioAutogradeCourse  `json:"course"`
}

type codioAutogradeStudent struct {
	Email    string `json:"email"`
	Id       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"fullName"`
}

type codioAutogradeCourse struct {
	Id         string                   `json:"id"`
	ProjectId  string                   `json:"projectId"`
	Lti        bool                     `json:"lti"`
	Assignment codioAutogradeAssignment `json:"assignment"`
}

type codioAutogradeAssignment struct {
	Id    string `json:"id"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type codioResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
