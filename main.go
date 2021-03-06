package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/html"
)

var sem = make(chan struct{}, 30)
var maxDepth = flag.Int("depth", 2, "the maximum depth of the websites we want to crawl")

// type Node struct {
// 	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

// 	Type      NodeType
// 	DataAtom  atom.Atom
// 	Data      string
// 	Namespace string
// 	Attr      []Attribute
// }

// type Attribute struct {
// 	Namespace, Key, Val string
// }

//NodeType
// const (
// 	ErrorNode NodeType = iota
// 	TextNode
// 	DocumentNode
// 	ElementNode
// 	CommentNode
// 	DoctypeNode
// 	// RawNode nodes are not returned by the parser, but can be part of the
// 	// Node tree passed to func Render to insert raw HTML (without escaping).
// 	// If so, this package makes no guarantee that the rendered HTML is secure
// 	// (from e.g. Cross Site Scripting attacks) or well-formed.
// 	RawNode
// )

type Website struct {
	URLlist []string
	depth   int
}

// Just for testing
func printEachHTMLNode(node *html.Node, depth int) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.ErrorNode:
		fmt.Println("ErrorNode")
	case html.TextNode:
		fmt.Println("TextNode")
	case html.DocumentNode:
		fmt.Println("DocumentNode")
	case html.ElementNode:
		fmt.Println("ElementNode")
	case html.CommentNode:
		fmt.Println("CommendNode")
	case html.DoctypeNode:
		fmt.Println("DoctypeNode")
	case html.RawNode:
		fmt.Println("RawNode")
	}

	fmt.Printf("%d %*.s %s\n", depth, depth, "", node.Data)
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		printEachHTMLNode(c, depth+1)
	}

}

func extractLinksFromHTMLNode(resp *http.Response, urlSlice *[]string, node *html.Node) {

	if node == nil {
		return
	}
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, val := range node.Attr {
			if val.Key != "href" {
				continue
			}
			link, err := resp.Request.URL.Parse(val.Val)
			if err != nil {
				continue
			}

			*urlSlice = append(*urlSlice, link.String())
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		extractLinksFromHTMLNode(resp, urlSlice, c)
	}

}

// This function  is used to extract links from URL, will call extractLinksFROMHTMLNODE
// to extract links from HTML node
func extractLinksFromURL(url string) ([]string, error) {
	sem <- struct{}{}

	fmt.Println(url)
	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTML status is not ok")
	}

	rootNode, err := html.Parse(resp.Body)

	if err != nil {
		return nil, err
	}

	urlSlice := []string{}
	extractLinksFromHTMLNode(resp, &urlSlice, rootNode)

	<-sem
	return urlSlice, nil
}

// Has to use channel to communicate between go routine
// Using BFS to crawl
func (w *Website) crawl() {

	rootURL := w.URLlist[0]
	urlList := make(chan Website)
	n := 0

	n++
	go func() {
		firstList, err := extractLinksFromURL(rootURL)
		if err != nil {
			log.Fatalln(err)
			urlList <- Website{[]string{}, *maxDepth + 1}
			return
		}
		urlList <- Website{firstList, w.depth + 1}
	}()

	seen := make(map[string]bool)
	for ; n > 0; n-- {
		parentWebsite := <-urlList
		//only spawn goroutine if the depth is smaller
		if parentWebsite.depth <= *maxDepth {
			// process parent list
			for _, val := range parentWebsite.URLlist {
				// add to childlist
				if !seen[val] {
					seen[val] = true
					n++
					go func(val string, innerDepth int) {
						childList, err := extractLinksFromURL(val)
						fmt.Println(innerDepth)
						if err != nil {
							urlList <- Website{[]string{}, *maxDepth + 1}
							fmt.Println(err)
							return
						}
						urlList <- Website{childList, innerDepth + 1}
					}(val, parentWebsite.depth)
				}
			}
		}
	}

}

func main() {

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println(flag.Args())
		log.Fatal("Please insert a valid url go run -depth=[depth] [url]")
	}
	targetWebsite := &Website{
		URLlist: []string{flag.Args()[0]},
		depth:   0,
	}
	targetWebsite.crawl()

}
