package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"

	"io"
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

func main() {

	target := "./index.html"

	var wg sync.WaitGroup
	// Open file
	file, err := os.Open(target)
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
		wg.Add(1)
		go worker(meta, &wg)
	}

	wg.Wait()

	replaceOriginFileWithInjectedAssets(doc)

	fmt.Printf("-----------------\nAssets injected into: %v\n-----------------\n", target)
	for _, a := range foundStyleFiles {
		fmt.Println(a.href)
	}

}

func worker(meta DOMAssetMeta, wg *sync.WaitGroup) {
	defer wg.Done()

	//TODO: Need to check if it's an external or internal link
	// and support that
	af, err := os.Open(meta.href)
	if err != nil {
		panic(err)
	}

	ct, err := ioutil.ReadAll(af)
	if err != nil {
		panic(err)
	}

	docInsertStyleNodeWithContent(&meta, string(ct))
	meta.node.Parent.RemoveChild(meta.node)
}

func replaceOriginFileWithInjectedAssets(doc *html.Node) error {
	newFile, err := os.Create("index.html")
	if err != nil {
		return err
	}
	fileBytes, err := nodeToBytes(doc)
	if err != nil {
		return err
	}
	newFile.Write(fileBytes)
	return nil
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
					// TODO: Remove duplicate assets
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
