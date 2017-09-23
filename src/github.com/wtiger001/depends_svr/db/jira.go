package db

import (
	"fmt"
	"log"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
)

type Search struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

type IssueList struct {
	Expand     string       `json:"expand"`
	StartAt    int          `json:"startAt"`
	MaxResults int          `json:"maxResults"`
	Total      int          `json:"total"`
	Issues     []jira.Issue `json:"issues"`
}

type JiraData struct {
	issues IssueList
	// sprints SprintList

}

// ExtractData contacts JIRA and extracts the contents into a database file
func ExtractData(cfg *JiraConfig) {
	// Setup Graph
	graph := NewGraph()

	// Set up the JIRA Client
	jiraClient, _ := jira.NewClient(nil, cfg.JiraURL)
	jiraClient.Authentication.SetBasicAuth(cfg.User, cfg.Password)

	// Load the issue types we consider static
	//loadStaticIssues(cfg, jiraClient, graph)

	// load the sprints
	loadBoards(cfg, jiraClient, graph)

	// save the database
	graph.save()
}

func loadBoards(cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	defer timeTrack(time.Now(), "Load Boards")
	startAt := 0
	pageSize := 100

	for {
		// Request the issue types that we consider static
		boardOpts := new(jira.BoardListOptions)
		boardOpts.ProjectKeyOrID = "PIR"
		boardOpts.StartAt = startAt
		boardOpts.MaxResults = pageSize
		boards, res, err := jiraClient.Board.GetAllBoards(boardOpts)
		if err != nil {
			log.Printf("%+v\n", res)
			log.Printf("Load Boards\n")
			panic(err)
		}

		// for _, board := range boards.Values {
		// 	log.Printf("\tBoard : %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)
		// 	// log.Printf("%+v\n", board)
		// }

		for _, board := range boards.Values {
			// log.Printf("\tBoard : %s\n", board.Name)
			// log.Printf("%+v\n", board)
			if board.Type == "kanban" {
				log.Printf("\tSkipping Board : %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)
			} else {
				log.Printf("\tBoard : %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)
				loadSprints(&board, cfg, jiraClient, graph)
			}
		}

		startAt = boards.StartAt + boards.MaxResults

		if boards.Total <= (boards.StartAt + boards.MaxResults) {
			log.Printf("Graph contains %d Items \n\n", len(graph.Items))
			return nil
		}
	}

}

func loadSprints(board *jira.Board, cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	sprints, res, err := jiraClient.Board.GetAllSprints(strconv.Itoa(board.ID))
	if err != nil {
		log.Printf(">>> %+v\n", res)
		log.Printf(">>> Load Sprints for Board %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)

		panic(err)
	}

	for _, sprint := range sprints {
		log.Printf("\t\tSprint : %s\n", sprint.Name)
		// loadSprint(&sprint, cfg, jiraClient, graph)
	}

	return nil
}

func loadSprint(sprint *jira.Sprint, cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	// Add the Sprint node
	graph.addSprint(sprint)

	// Get the issues
	issues, res, err := jiraClient.Sprint.GetIssuesForSprint(sprint.ID)
	if err != nil {
		log.Printf("%+v\n", res)
		log.Printf("Load Sprint %s\n", sprint.Name)
		panic(err)
	}

	// Aggregate the issues
	for _, issue := range issues {
		aggregateSprintIssue(sprint, &issue, cfg, graph)
	}
	return nil
}

func aggregateSprintIssue(sprint *jira.Sprint, issue *jira.Issue, cfg *JiraConfig, graph *Graph) {

}

func loadStaticIssues(cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	defer timeTrack(time.Now(), "Load Static Issues")
	startAt := 0
	pageSize := 100
	for {
		// Request the issue types that we consider static
		issues, err := requestStaticIssues(cfg, jiraClient, startAt, pageSize)
		if err != nil {
			panic(err)
		}
		if startAt == 0 {
			fmt.Printf("Identifed Total %d Issues\n\n", issues.Total)
		}
		graph.addStatic(issues)

		startAt = issues.StartAt + issues.MaxResults

		if issues.Total <= (issues.StartAt + issues.MaxResults) {
			log.Printf("Graph contains %d Items \n\n", len(graph.Items))
			return nil
		}
	}
}

func requestStaticIssues(cfg *JiraConfig, jiraClient *jira.Client, startAt int, pageSize int) (issues *IssueList, err error) {
	defer timeTrack(time.Now(), "Request Static Issues")

	// Create the Search Body
	opts := new(Search)
	opts.JQL = makeJql(cfg)
	opts.StartAt = startAt
	opts.MaxResults = pageSize
	opts.Fields = []string{"summary", "issuetype", "status", "components", "labels", "issuelinks"}

	req, _ := jiraClient.NewRequest("POST", "rest/api/2/search", opts)

	// Save a copy of this request for debugging.
	if cfg.Debug {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		fmt.Println(string(requestDump))
	}

	// Exectute the request
	issues = new(IssueList)
	_, err = jiraClient.Do(req, issues)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func TestJira() {
	//
	jiraClient, _ := jira.NewClient(nil, "https://issues.apache.org/jira/")
	issue, _, _ := jiraClient.Issue.Get("MESOS-3325", nil)

	fmt.Printf("%s: %+v\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("Type: %s\n", issue.Fields.Type.Name)
	fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)
}

func makeJql(cfg *JiraConfig) string {
	// "project = PIR AND issuetype in ('New Capability', 'New Feature', 'Requirement', 'Thread')"

	jql := ""

	// Project
	if len(cfg.Projects) == 1 {
		jql = "project = " + cfg.Projects[0]
	} else {
		jql = "project IN (" + strings.Join(cfg.Projects[:], ",") + ")"
	}

	// AND
	jql += " AND "

	// Types
	types := []string{cfg.CapabilityIssueType, cfg.FeatureIssueType, cfg.RequirementIssueType, cfg.ThreadIssueType}
	jql += "issuetype in ('" + strings.Join(types[:], "','") + "')"

	return jql
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
