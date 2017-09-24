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

	// Load the components
	loadComponents(cfg, jiraClient, graph)

	// Load the issue types we consider static
	loadStaticIssues(cfg, jiraClient, graph)

	// load the sprints
	loadBoards(cfg, jiraClient, graph)

	// graph.addMissingNodes(cfg)
	// graph.trimMissing(cfg)

	// save the database
	graph.save(cfg)
}

func loadComponents(cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	defer timeTrack(time.Now(), "Load Components")

	components, err := requestComponents(cfg.Projects[0], cfg, jiraClient)
	if err != nil {
		panic(err)
	}

	for _, component := range components {
		graph.componentNode(component.Name, component.Description)
	}

	return nil
}

func requestComponents(projectKey string, cfg *JiraConfig, jiraClient *jira.Client) (components []jira.ProjectComponent, err error) {
	defer timeTrack(time.Now(), "Request Components")

	url := fmt.Sprintf("rest/api/2/project/%s/components", projectKey)
	req, _ := jiraClient.NewRequest("GET", url, nil)

	// Save a copy of this request for debugging.
	if cfg.Debug {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		fmt.Println(string(requestDump))
	}

	// Exectute the request
	items := new([]jira.ProjectComponent)
	_, err = jiraClient.Do(req, items)
	if err != nil {
		return nil, err
	}

	return *items, nil
}

func loadBoards(cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (err error) {
	sprintMap, _ := getBoards(cfg, jiraClient, graph)

	log.Printf("%d Scrum Sprints Found\n", len(sprintMap))
	for _, v := range sprintMap {
		loadSprint(&v, cfg, jiraClient, graph)
	}
	return nil
}

func getBoards(cfg *JiraConfig, jiraClient *jira.Client, graph *Graph) (sprintMap map[int]jira.Sprint, err error) {
	defer timeTrack(time.Now(), "Get Boards")
	startAt := 0
	pageSize := 100
	sprintMap = make(map[int]jira.Sprint)

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

		for _, board := range boards.Values {
			// log.Printf("\tBoard : %s\n", board.Name)
			// log.Printf("%+v\n", board)
			if board.Type == "kanban" {
				if cfg.Debug {
					log.Printf("\tSkipping Board : %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)
				}
			} else {
				if cfg.Debug {
					log.Printf("\tBoard : %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)
				}
				mapSprints(&board, cfg, jiraClient, graph, sprintMap)
			}
		}

		startAt = boards.StartAt + boards.MaxResults

		if boards.Total <= (boards.StartAt + boards.MaxResults) {
			log.Printf("Graph contains %d Items \n\n", len(graph.Items))
			return sprintMap, nil
		}
	}

}

func mapSprints(board *jira.Board, cfg *JiraConfig, jiraClient *jira.Client, graph *Graph, sprintMap map[int]jira.Sprint) (err error) {
	defer timeTrack(time.Now(), "Map Sprints")

	sprints, res, err := jiraClient.Board.GetAllSprints(strconv.Itoa(board.ID))
	if err != nil {

		log.Printf(">>> %+v\n", res)
		log.Printf(">>> Load Sprints for Board %s Type: %s ID: %d\n", board.Name, board.Type, board.ID)

		panic(err)
	}

	for _, sprint := range sprints {
		if cfg.Debug {
			log.Printf("\t\tSprint : %s\n", sprint.Name)
		}
		sprintMap[sprint.ID] = sprint
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
		panic(err)
	}

	// Aggregate the issues
	log.Printf("Loading %d issues for Sprint %s\n", len(issues), sprint.Name)
	for _, issue := range issues {
		aggregateSprintIssue(sprint, &issue, cfg, graph)
	}
	return nil
}

func getDependencyType(issueType string, cfg *JiraConfig) string {
	switch issueType {
	case cfg.CapabilityIssueType:
		return cfg.DependsLinkOut
	case cfg.FeatureIssueType:
		return cfg.DependsLinkOut
	case cfg.RequirementIssueType:
		return cfg.TracesToLink
	case cfg.ThreadIssueType:
		return cfg.DependsLinkOut
	}
	return cfg.DependsLinkOut
}

func trackedLinkType(linkType string, cfg *JiraConfig) bool {
	switch strings.ToLower(linkType) {
	case strings.ToLower(cfg.ChildLink):
		return true
	case strings.ToLower(cfg.ParentLink):
		return true
	case strings.ToLower(cfg.TracesFromLink):
		return true
	case strings.ToLower(cfg.TracesToLink):
		return true
	case strings.ToLower(cfg.DependsLinkIn):
		return true
	case strings.ToLower(cfg.DependsLinkOut):
		return true
	}
	return false
}

// aggregateSprintIssue uses the imformation from each issue to determine dependancies on the
// overall sprint node. The dependency types are Process, Component, Feature, Capability and Requirement
func aggregateSprintIssue(sprint *jira.Sprint, issue *jira.Issue, cfg *JiraConfig, graph *Graph) {
	// Create direct dependency
	switch issue.Fields.Type.Name {
	case cfg.CapabilityIssueType:
		fallthrough
	case cfg.FeatureIssueType:
		fallthrough
	case cfg.RequirementIssueType:
		fallthrough
	case cfg.ThreadIssueType:
		linkID := validID(issue.Key + "_SPRINT_" + strconv.Itoa(sprint.ID))
		if edge, ok := graph.m[linkID]; !ok {
			edge := Edge()
			edge.Data.Target = validID(issue.Key)
			edge.Data.Source = validID(strconv.Itoa(sprint.ID))
			edge.Data.Id = linkID
			edge.Data.Description = fmt.Sprintf("Issue %s", issue.Key)
			// edge.Data.Type = getDependencyType(issueType, cfg)
			edge.Data.Type = getDependencyType(issue.Fields.Type.Name, cfg)
			graph.add(edge)
		} else {
			edge.Data.Description += fmt.Sprintf("\nIssue %s", issue.Key)
		}
	}

	// Navigate the links
	for _, link := range issue.Fields.IssueLinks {
		// Determine what is being linked to
		linked, issueType, linkType, _ := linked(link)
		if trackedLinkType(linkType, cfg) {
			switch issueType {
			// Create direct dependency
			case cfg.CapabilityIssueType:
				fallthrough
			case cfg.FeatureIssueType:
				fallthrough
			case cfg.RequirementIssueType:
				fallthrough
			case cfg.ThreadIssueType:
				linkID := validID(linked.ID + "_SPRINT_" + strconv.Itoa(sprint.ID))
				if edge, ok := graph.m[linkID]; !ok {
					edge := Edge()
					edge.Data.Target = validID(linked.Key)
					edge.Data.Source = validID(strconv.Itoa(sprint.ID))
					edge.Data.Id = linkID
					edge.Data.Description = fmt.Sprintf("Issue %s link %s", issue.Key, linked.Key)
					// edge.Data.Type = getDependencyType(issueType, cfg)
					edge.Data.Type = linkType
					graph.add(edge)
				} else {
					edge.Data.Description += fmt.Sprintf("\nIssue %s process label %s", issue.Key, linked.Key)
				}
			default:
				// Capture aggregated dependency
				if cfg.Debug {
					log.Printf("\tUnsure how to capture link between %s (%s) and %s (%s) of type %s\n", issue.Key, issue.Fields.Type.Name, linked.Key, linked.Fields.Type.Name, linkType)
				}
			}
		}
	}

	// Read the Labels
	for _, label := range issue.Fields.Labels {
		if strings.HasPrefix(strings.ToLower(label), strings.ToLower(cfg.ProcessPrefix)) {
			// Extract the process name
			process := strings.TrimPrefix(strings.ToLower(label), strings.ToLower(cfg.ProcessPrefix))

			linkID := validID(process + "_SPRINT_" + strconv.Itoa(sprint.ID))
			if edge, ok := graph.m[linkID]; !ok {
				edge := Edge()
				edge.Data.Target = validID(process)
				edge.Data.Source = validID(strconv.Itoa(sprint.ID))
				edge.Data.Id = linkID
				edge.Data.Description = fmt.Sprintf("Issue %s process label %s", issue.Key, label)
				edge.Data.Type = cfg.DependsLinkOut
				graph.add(edge)
			} else {
				edge.Data.Description += fmt.Sprintf("\nIssue %s process label %s", issue.Key, label)
			}
		}
	}
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
		graph.addStatic(issues, cfg)

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
	opts.Fields = []string{"summary", "issuetype", "status", "components", "labels", "issuelinks", "description", "customfield_13008"}

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

	//

	return jql
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func linked(link *jira.IssueLink) (linked *jira.Issue, issueType string, linkType string, direction int) {
	if link.InwardIssue != nil {
		return link.InwardIssue, link.InwardIssue.Fields.Type.Name, link.Type.Inward, 1
	}
	if link.OutwardIssue != nil {
		return link.OutwardIssue, link.OutwardIssue.Fields.Type.Name, link.Type.Outward, -1
	}
	return nil, "", "", 0
}

func issueType(link *jira.IssueLink) string {
	if link.InwardIssue != nil {
		return link.InwardIssue.Fields.Type.Name
	}
	if link.OutwardIssue != nil {
		return link.OutwardIssue.Fields.Type.Name
	}
	return ""
}
