package htmlcrawler

import (
	"golang.org/x/net/html"
)

func CrawlByTag(tagName string, node *html.Node) *html.Node {
	if node.Type == html.ElementNode && node.Data == tagName {
		return node
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if node := CrawlByTag(tagName, c); node != nil {
			return node
		}
	}

	return nil
}

func CrawlByTagAll(tagName string, node *html.Node) []*html.Node {
	var nodes []*html.Node

	if node.Type == html.ElementNode && node.Data == tagName {
		nodes = append(nodes, node)
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if node := CrawlByTag(tagName, c); node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func GetNodeAttributes(node *html.Node) map[string]string {
	attributes := make(map[string]string)

	for _, attribute := range node.Attr {
		attributes[attribute.Key] = attribute.Val
	}

	return attributes
}
