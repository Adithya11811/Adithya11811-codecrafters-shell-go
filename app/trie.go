package main


type TrieNode struct {
	Children map[rune]*TrieNode
	IsEnd     bool
}

func NewTrieNode() *TrieNode {
	return &TrieNode{
		Children: make(map[rune]*TrieNode),
		IsEnd:    false,
	}
}

type Trie struct {
	Root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		Root: NewTrieNode(),
	}
}

func (t* Trie) insert(word string) {
	node := t.Root
	for _, char := range word {
		if _, exists := node.Children[char]; !exists {
			node.Children[char] = NewTrieNode()
		}
		node = node.Children[char]
	}
	node.IsEnd = true
}

func (t *Trie) search(word string) bool {
	node := t.Root
	for _, char := range word {
		if _, exists := node.Children[char]; !exists {
			return false
		}
		node = node.Children[char]
	}
	return node.IsEnd
}

func (t *Trie) AutoComplete(prefix string) []string {
	var results []string
	node := t.Root
	for _, ch := range prefix {
		if node.Children[ch] == nil {
			return results
		}
		node = node.Children[ch]
	}
	t.dfs(node, prefix, &results)
	return results
}

func (t* Trie) dfs(node *TrieNode, prefix string, results *[]string) {
	if node.IsEnd {
		*results = append(*results, prefix)
	}
	for char, child := range node.Children {
		t.dfs(child, prefix+string(char), results)
	}
}
