package main

import (
	"bytes"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// DOMAssetMeta is meta data about the file
type DOMAssetMeta struct {
	node *html.Node
	href string
}

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

	for _, meta := range foundStyleFiles {
		docInsertStyleNodeWithContent(&meta, ".foo { color: blue }")
		meta.node.Parent.RemoveChild(meta.node)
	}

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

// findAllStyleAssetPaths finds all links for stylesheets returns an slice
// with their DOM node and href value
func findAllStyleAssetPaths(doc *html.Node) ([]DOMAssetMeta, error) {

	assetsFound := []DOMAssetMeta{}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			for _, a := range n.Attr {
				if strings.Contains(a.Val, ".css") {
					assetsFound = append(assetsFound, DOMAssetMeta{
						n, a.Val,
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
