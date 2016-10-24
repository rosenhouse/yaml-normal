package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Key string
type Value string

type Relation int

const (
	Unknown Relation = iota
	Unrelated
	LeftDerivedFromRight
	RightDerivedFromLeft
	IsEqualTo
)

type ValuesMap map[Key]Value

func (raw ValuesMap) findRelation(key1, key2 Key) Relation {
	v1, v2 := string(raw[key1]), string(raw[key2])
	switch {
	case v1 == v2:
		return IsEqualTo
	case strings.Contains(v1, v2):
		return LeftDerivedFromRight
	case strings.Contains(v2, v1):
		return RightDerivedFromLeft
	default:
		return Unrelated
	}
}

type GraphNode struct {
	Name string `json:"name"`
}

type GraphLink struct {
	Source int `json:"source"`
	Target int `json:"target"`
}

type FinalGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

func (values ValuesMap) buildGraph() FinalGraph {
	var nodes []GraphNode
	var links []GraphLink

	for key, _ := range values {
		nodes = append(nodes, GraphNode{Name: string(key)})
	}

	for index1, node1 := range nodes {
		for index2, node2 := range nodes {
			switch values.findRelation(Key(node1.Name), Key(node2.Name)) {
			case IsEqualTo, LeftDerivedFromRight:
				links = append(links, GraphLink{Source: index1, Target: index2})
			}
		}
	}

	return FinalGraph{Nodes: nodes, Links: links}
}

func mainWithError() error {
	var inPath string

	flag.StringVar(&inPath, "input", "", "file to parse")
	flag.Parse()

	if inPath == "" {
		return fmt.Errorf("missing required flag input")
	}

	inBytes, err := ioutil.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading input: %s", err)
	}

	var valuesMap ValuesMap
	if err := yaml.Unmarshal(inBytes, &valuesMap); err != nil {
		return fmt.Errorf("parsing yaml: %s", err)
	}

	finalGraph := valuesMap.buildGraph()
	graphJSON, err := json.Marshal(finalGraph)
	if err != nil {
		return fmt.Errorf("marshal graph to json: %s", err)
	}

	templateBytes, err := ioutil.ReadFile("template.html")
	if err != nil {
		return fmt.Errorf("reading template: %s", err)
	}

	generatedBytes := bytes.Replace(templateBytes, []byte("REPLACE_ME"), graphJSON, 1)

	outFile, err := ioutil.TempFile("", "demo")
	if err != nil {
		return fmt.Errorf("creating temp file: %s", err)
	}
	if _, err := outFile.Write(generatedBytes); err != nil {
		return fmt.Errorf("writing generated file: %s", err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("closing generated file: %s", err)
	}

	if !startBrowser("file://" + outFile.Name()) {
		fmt.Fprintf(os.Stderr, "HTML output written to %s\n", outFile.Name())
	}

	return nil
}

func main() {
	if err := mainWithError(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

// startBrowser tries to open the URL in a browser
// and reports whether it succeeds.
func startBrowser(url string) bool {
	// try to start the browser
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	return cmd.Start() == nil
}
