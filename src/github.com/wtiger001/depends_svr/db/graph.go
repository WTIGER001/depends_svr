package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
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
	typeSource  string
	typeTarget  string
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
	item.Data.Id = validID(item.Data.Id)
	item.Data.Target = validID(item.Data.Target)
	item.Data.Source = validID(item.Data.Source)

	if _, ok := graph.m[item.Data.Id]; ok {
		log.Printf("Duplicate nodes with key " + item.Data.Id)
	} else {
		graph.m[item.Data.Id] = item
		graph.Items = append(graph.Items, item)
	}
}

func (graph *Graph) addSprint(sprint *jira.Sprint) {
	n := Node()
	n.Data.Id = strconv.Itoa(sprint.ID)
	n.Data.Label = sprint.Name
	if sprint.StartDate != nil {
		n.Data.StartDate = sprint.StartDate.String()
	}
	if sprint.EndDate != nil {
		n.Data.FinishDate = sprint.EndDate.String()
	}
	n.Data.Status = sprint.State
	n.Data.Type = "Sprint"
	graph.add(n)
}

func getEdgeType(linkType string, cfg *JiraConfig) string {
	return linkType
}

func getNodeType(nodeType string, cfg *JiraConfig) string {
	switch nodeType {
	case cfg.CapabilityIssueType:
		return "capability"
	case cfg.FeatureIssueType:
		return "feature"
	case cfg.ThreadIssueType:
		return "thread"
	case cfg.RequirementIssueType:
		return "requirement"
	default:
		return nodeType
	}
}

func supportedIssueType(issueType string, cfg *JiraConfig) bool {
	real := getNodeType(issueType, cfg)
	switch real {
	case "capability":
		return true
	case "feature":
		return true
	case "thread":
		return true
	case "requirement":
		return true
	default:
		return false
	}
}

func (graph *Graph) addStatic(issues *IssueList, cfg *JiraConfig) {
	defer timeTrack(time.Now(), "Add Static Nodes")

	cntNodes := 0
	cntEdges := 0
	// Iterate through each issue and create nodes
	for _, issue := range issues.Issues {

		// Create the node
		n := Node()
		n.Data.Id = validID(issue.Key)
		n.Data.Label = issue.Fields.Summary
		n.Data.Description = issue.Fields.Description
		n.Data.Component = first(issue.Fields.Components)
		n.Data.Type = getNodeType(issue.Fields.Type.Name, cfg)

		if n.Data.Type == "thread" {
			a := issue.Fields.Unknowns["customfield_13008"]
			if a != nil {
				n.Data.FinishDate = a.(string)
			}
		}

		cntNodes++
		graph.add(n)

		// Create the link to the components
		for _, c := range issue.Fields.Components {
			graph.componentLink(c.Name, n, cfg)
		}

		// Create any links
		for _, link := range issue.Fields.IssueLinks {
			_, issueType, linkType, _ := linked(link)
			if trackedLinkType(linkType, cfg) && supportedIssueType(issueType, cfg) {
				e := Edge()
				e.Data.Id = validID(link.ID)

				if _, ok := graph.m[e.Data.Id]; !ok {
					// Duplicates are OK since links are Bi-Directional
					if link.OutwardIssue != nil {
						e.Data.Source = link.OutwardIssue.Key
						e.Data.typeSource = getNodeType(link.OutwardIssue.Fields.Type.Name, cfg)
					} else {
						e.Data.Source = n.Data.Id
						e.Data.typeSource = n.Data.Type
					}
					if link.InwardIssue != nil {
						e.Data.Target = validID(link.InwardIssue.Key)
						e.Data.typeTarget = getNodeType(link.InwardIssue.Fields.Type.Name, cfg)
					} else {
						e.Data.Target = n.Data.Id
						e.Data.typeTarget = n.Data.Type
					}

					e.Data.Type = getEdgeType(linkType, cfg)
					if link.Comment != nil {
						e.Data.Description = link.Comment.Body
					}
					cntEdges++
					graph.add(e)
				}
			}
		}
	}
	fmt.Printf("Added static: %d Nodes and %d Edges\n", cntNodes, cntEdges)
}

func first(components []*jira.Component) string {
	if len(components) >= 1 {
		return components[0].Name
	}
	return ""
}

func (graph *Graph) save(cfg *JiraConfig) (err error) {
	return graph.saveAs(cfg.OutputFile)
}

func (graph *Graph) saveAs(file string) (err error) {
	defer timeTrack(time.Now(), "Save Output as "+file)

	graph.printSummary()

	sort.Slice(graph.Items, func(i, j int) bool {
		return graph.Items[i].Group > graph.Items[j].Group
	})

	graphJSON, _ := json.Marshal(graph)
	err = ioutil.WriteFile(file, graphJSON, 0644)

	log.Printf("Wrote %s file for database containing %d nodes and edges\n", file, len(graph.Items))

	return err
}

func (graph *Graph) checkForMissing() {
	ok := 0
	bad := 0
	log.Printf("Checking Node Structure\n")
	log.Printf("-----------------------------------------\n")
	for _, item := range graph.Items {
		if item.Group == "edges" {
			if item.Data.Target == "" {
				log.Printf("\tEmpty Target\n")
				bad++
			} else if item.Data.Source == "" {
				log.Printf("\tEmpty Source\n")
				bad++
			} else if !graph.exists(item.Data.Target) {
				log.Printf("\tMising Target Node: %s (%s)\n", item.Data.Target, item.Data.typeTarget)
				bad++
			} else if !graph.exists(item.Data.Target) {
				log.Printf("\tMising Source Node: %s (%s)\n", item.Data.Source, item.Data.typeSource)
				bad++
			} else {
				ok++
			}
		}
	}
	log.Printf("-----------------------------------------\n")
	log.Printf("Good: %d, Bad: %d\n", ok, bad)
}

func (graph *Graph) addMissingNodes(cfg *JiraConfig) {
	// Add all the missing nodes
	for _, item := range graph.Items {
		if item.Group == "edges" {
			if item.Data.Target != "" && item.Data.Source != "" {
				if !graph.exists(item.Data.Target) {
					log.Printf("Mising Target Node: " + item.Data.Target)
					graph.guessNode(item.Data.typeTarget, item.Data.Target)
				}
				if !graph.exists(item.Data.Source) {
					log.Printf("Mising Source Node: " + item.Data.Target)
					graph.guessNode(item.Data.typeSource, item.Data.Source)
				}
			}
		}
	}
}

func (graph *Graph) trimMissing(cfg *JiraConfig) {
	del := *new([]int)
	// Add all the missing nodes
	for i, item := range graph.Items {
		if item.Group == "edges" {
			if item.Data.Target != "" && item.Data.Source != "" {
				if !graph.exists(item.Data.Target) {
					del = append(del, i)
				} else if !graph.exists(item.Data.Source) {
					del = append(del, i)
				}
			}
		}
	}

	for i := len(del) - 1; i >= 0; i-- {
		graph.Items = append(graph.Items[:i], graph.Items[i+1:]...)
	}
}

func (graph *Graph) guessNode(expectType string, id string) {
	n := Node()
	n.Data.Id = validID(id)
	n.Data.Label = id
	n.Data.Type = expectType
}

func (graph *Graph) componentLink(component string, n *GraphItem, cfg *JiraConfig) {
	cNode := graph.componentNode(component, "")

	if cNode == nil || n == nil {
		log.Printf("WHATTTTTT ")
	} else {
		e := Edge()
		e.Data.Id = n.Data.Id + "_COMPONENT_" + cNode.Data.Id
		e.Data.Target = cNode.Data.Id
		e.Data.Source = n.Data.Id
		e.Data.Type = cfg.DependsLinkOut
		graph.add(e)
	}
}

func (graph *Graph) componentNode(component string, desc string) *GraphItem {
	real := validID(component)
	if !graph.exists(real) {
		n := Node()
		n.Data.Id = real
		n.Data.Label = component
		n.Data.Description = desc
		n.Data.Type = "component"
		graph.add(n)
	}
	return graph.m[real]
}

func (graph *Graph) exists(k string) bool {
	_, ok := graph.m[k]
	return ok
}

func (graph *Graph) printSummary() {
	nodes, edges := graph.histogram()

	sumNodes := 0
	sumEdges := 0
	for _, v := range nodes {
		sumNodes += v
	}
	for _, v := range edges {
		sumEdges += v
	}

	log.Printf("GRAPH SUMMARY\n")
	log.Printf("-----------------------------------------\n")
	log.Printf("%-33s:%6d\n", "Nodes", sumNodes)
	for k, v := range nodes {
		log.Printf("   %-30s:%6d\n", k, v)
	}
	log.Printf("%-33s:%6d\n", "Edges", sumEdges)
	for k, v := range edges {
		log.Printf("   %-30s:%6d\n", k, v)
	}
	log.Printf("-----------------------------------------\n")

	graph.checkForMissing()
}

func validID(id string) string {
	newID := strings.Replace(id, " ", "_", -1)
	return newID
}

func (graph *Graph) histogram() (nodes map[string]int, edges map[string]int) {
	nodes = make(map[string]int)
	edges = make(map[string]int)

	for _, item := range graph.Items {
		key := item.Data.Type
		if item.Group == "edges" {
			v := edges[key]
			v++
			edges[key] = v
		} else {
			v := nodes[key]
			v++
			nodes[key] = v
		}

	}
	return nodes, edges
}
