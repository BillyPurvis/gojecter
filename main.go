package main

import (
	"bytes"
	
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// DOMAssetMeta is meta data about the file
type DOMAssetMeta struct {
	node *html.Node
	href string
}

// Queue is a queue of DOMAssetMeta
type Queue chan DOMAssetMeta

var toRetrieve Queue = make(chan DOMAssetMeta)

func main() {
	// Open file
	file, err := os.Open("./index.html")
	if err != nil {
		panic(err)
	}

	doc, err := html.Parse(file)
	if err != nil {
		panic(err)
	}

	foundStyleFiles, err := findAllStyleAssetPaths(doc)
	if err != nil {
		panic(err)
	}

	// Chuck found assets into Queue
	go func(c chan DOMAssetMeta) {
		for _, meta := range foundStyleFiles {
			c <- meta
		}
	}(toRetrieve)

	// Read from it
	go getFileContents(toRetrieve)

	// TODO: The system exits before the routine is fished. Need to use syncGroup

	
	newFile, err := os.Create("index.html")
	if err != nil {
		panic(err)
	}

	fileBytes, err := nodeToBytes(doc)
	if err != nil {
		panic(err)
	}
	newFile.Write(fileBytes)
}

func getFileContents(c chan DOMAssetMeta) {
	for {
		f, ok := <-c
		if !ok {
			return
		}		
		af, err := os.Open(f.href)
		if err != nil {
			panic(err)
		}
		
		ct, err := ioutil.ReadAll(af)
		if err != nil {
			panic(err)
		}
	
		docInsertStyleNodeWithContent(&f, string(ct))
		f.node.Parent.RemoveChild(f.node)
		
	}
}

func trimQueryStrFromHref(s string) (string, error) {
	r, err := regexp.Compile("\\?.*")
	if err != nil {
		return "", err
	}
	s = r.ReplaceAllString(s, "")
	return s, nil
}

// findAllStyleAssetPaths finds all links for stylesheets returns an slice
// with their DOM node and href value
func findAllStyleAssetPaths(doc *html.Node) ([]DOMAssetMeta, error) {
	assetsFound := []DOMAssetMeta{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			for _, a := range n.Attr {
				if strings.Contains(a.Val, ".css") {

					hrefTrim, err := trimQueryStrFromHref(a.Val)
					if err != nil {
						panic(err)
					}

					assetsFound = append(assetsFound, DOMAssetMeta{
						n, hrefTrim,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return assetsFound, nil
}

// Insert a new style tag replacing the linked stylesheet with the stylesheets content
func docInsertStyleNodeWithContent(n *DOMAssetMeta, content string) {
	styleNode := &html.Node{
		Type: html.ElementNode,
		Data: "style",
		FirstChild: &html.Node{
			Type: html.TextNode,
			Data: content,
		},
	}
	n.node.InsertBefore(styleNode, n.node)
}

// Transform *html.Node into a byte slice
func nodeToBytes(n *html.Node) ([]byte, error) {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	err := html.Render(w, n)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}