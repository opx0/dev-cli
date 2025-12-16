package llm

type Step struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	File    string `json:"file,omitempty"`
	Note    string `json:"note,omitempty"`
}

type Solution struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
	Source      string `json:"source,omitempty"`
}

type ResearchResult struct {
	Query     string     `json:"query"`
	Solutions []Solution `json:"solutions"`
}

type LogAnalysisResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}
