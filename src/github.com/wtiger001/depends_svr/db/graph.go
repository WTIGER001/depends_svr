package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

type Graph struct {
	Items []*GraphItem `json:"graph"`
	m     map[string]*GraphItem
}

type GraphItem struct {
	Group string `json:"group"`
	Data  *Data  `json:"data"`
}

type Data struct {
	Id          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Parent      string `json:"parent,omitempty"`
	Source      string `json:"source,omitempty"`
	Target      string `json:"target,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	Type        string `json:"type,omitempty"`
	Degree      int    `json:"degree,omitempty"`
	Version     string `json:"version,omitempty"`
	Component   string `json:"component,omitempty"`
	Status      string `json:"status,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	FinishDate  string `json:"finish_date,omitempty"`
	Description string `json:"description,omitempty"`
}

func Node() *GraphItem {
	g := new(GraphItem)
	g.Group = "nodes"
	g.Data = new(Data)

	return g
}

func Edge() *GraphItem {
	g := new(GraphItem)
	g.Group = "edges"
	g.Data = new(Data)

	return g
}

func NewGraph() *Graph {
	g := new(Graph)
	g.m = make(map[string]*GraphItem)
	return g
}

func (graph *Graph) add(item *GraphItem) {
	if _, ok := graph.m[item.Data.Id]; ok {
		log.Printf("Duplicate nodes with key " + item.Data.Id)
	} else {
		graph.m[item.Data.Id] = item
		graph.Items = append(graph.Items, item)
	}
}

func (graph *Graph) addStatic(issues *IssueList) {
	defer timeTrack(time.Now(), "Add Static Nodes")

	cntNodes := 0
	cntEdges := 0
	// Iterate through each issue and create nodes
	for _, issue := range issues.Issues {

		// Create the node
		n := Node()
		n.Data.Id = issue.Key
		n.Data.Label = issue.Fields.Summary
		n.Data.Description = issue.Fields.Description
		n.Data.Component = first(issue.Fields.Components)
		cntNodes++
		graph.add(n)

		// Create any links (dont enforce)
		for _, link := range issue.Fields.IssueLinks {
			e := Edge()
			e.Data.Id = link.ID

			if _, ok := graph.m[e.Data.Id]; !ok {
				// Duplicates are OK since links are Bi-Directional
				if link.OutwardIssue != nil {
					e.Data.Source = link.OutwardIssue.Key
				} else {
					e.Data.Source = n.Data.Id
				}
				if link.InwardIssue != nil {
					e.Data.Target = link.InwardIssue.Key
				} else {
					e.Data.Target = n.Data.Id
				}

				e.Data.Type = link.Type.Name
				if link.Comment != nil {
					e.Data.Description = link.Comment.Body
				}
				cntEdges++
				graph.add(e)
			}
		}
	}
	fmt.Printf("Added static: %d Nodes and %d Edges\n", cntNodes, cntEdges)
}

func first(components []*jira.Component) string {
	if len(components) >= 1 {
		return components[0].ID
	}
	return ""
}

func (graph *Graph) save() (err error) {
	return graph.saveAs("output.json")
}

func (graph *Graph) saveAs(file string) (err error) {
	defer timeTrack(time.Now(), "Save Output as "+file)
	graphJson, _ := json.Marshal(graph)
	err = ioutil.WriteFile(file, graphJson, 0644)
	return err
}
