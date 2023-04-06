package main

type gradescopeSubmission struct {
	Id                  string                         `json:"id"`
	CreatedAt           string                         `json:"created_at"`
	Assignment          gradescopeSubmissionAssignment `json:"assignment"`
	SubmissionMethod    string                         `json:"submission_method"`
	Users               []gradescopeSubmissionUser     `json:"users"`
	PreviousSubmissions []gradescopeSubmissionPrevious `json:"previous_submissions"`
}

type gradescopeSubmissionPrevious struct {
	SubmissionTime string  `json:"submission_time"`
	Score          float64 `json:"score"`
	Results        string  `json:"results"`
}

type gradescopeSubmissionUser struct {
	Email string `json:"email"`
	Id    string `json:"id"`
	Name  string `json:"name"`
}

type gradescopeSubmissionAssignment struct {
	DueDate         string `json:"due_date"`
	GroupSize       int    `json:"group_size"`
	GroupSubmission bool   `json:"group_submission"`
	Id              string `json:"id"`
	CourseId        string `json:"course_id"`
	LateDueDate     string `json:"late_due_date"`
	ReleaseDate     string `json:"release_date"`
	Title           string `json:"title"`
	TotalPoints     string `json:"total_points"`
}

type gradescopeResult struct {
	Score            float64                       `json:"score"`
	ExecutionTime    int                           `json:"execution_time"`
	Output           string                        `json:"output"`
	OutputFormat     string                        `json:"output_format"`
	TestOutputFormat string                        `json:"test_output_format"`
	TestNameFormat   string                        `json:"test_name_format"`
	Visibility       string                        `json:"visibility"`
	StdoutVisibility string                        `json:"stdout_visibility"`
	ExtraData        any                           `json:"extra_data"`
	Tests            []gradescopeResultTests       `json:"tests"`
	Leaderboard      []gradescopeResultLeaderboard `json:"leaderboard"`
}
type gradescopeResultTests struct {
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Status   string  `json:"status"` // optional, see "Test case status" below
	Name     string  `json:"name"`   // optional
	// "name_format": "text", // optional formatting for the test case name, see "Output String Formatting" below
	// "number": "1.1", // optional (will just be numbered in order of array if no number given)
	Output string `json:"output"`
	// "output_format": "text", // optional formatting for the test case output, see "Output String Formatting" below
	// "tags": ["tag1", "tag2", "tag3"], // optional
	Visibility string `json:"visibility"` // Optional visibility setting
	// "extra_data": {} // Optional extra data to be stored
}

type gradescopeResultLeaderboard struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Order string `json:"order"`
}
