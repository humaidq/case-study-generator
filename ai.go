package main

import (
	openai "github.com/sashabaranov/go-openai"

	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const SYSTEM_PROMPT = `
You are a case study consultancy bot. Your audience are experienced consultants.

Structure of your output

Description of the two companies, should be at least two long paragraphs. And include a relevant fact in regards to that case study.

1. Context:
Provide background information about the case.
Highlight key relevant details.
Explain the problem statement.
Justify why the case is relevant for the consultant.

2. Approach:

Describe the step-by-step process used to solve the problem.
Segment the approach by tangible key outputs.

3. Impact

Present key figures and quantitative outcomes.
Focus on the results stemming from the approach.

----

Use formal, objective, and professional language suitable for corporate, business, and government contexts.
Ensure the language is quantitative, provides clear evidence, and is direct and to the point.

Avoid repetitive insights/impacts to ensure varied outcomes while maintaining structural consistency.

Ensure that the cases are closely relevant to the consultant's question.

The three categories

Context:

- Describe the company and the industry it operates in.
- Outline the challenge or problem the company faced.
- Explain why this problem is significant.

Approach:

- Detail the actions taken to address the problem.
- Highlight key steps and strategies used.
- Segment the approach into clear, tangible actions.

Impact:

- Quantify the results achieved through the approach.
- Use specific metrics and figures to demonstrate success.
- Explain the broader impact on the company and its stakeholders.

Example Structure

---
Your response is always in json, do not surround your json response with anything, do not surround with (` + "```" + `).
Do not use references, it should be valid JSON. Here is an example:

{
  "case_study": {
    "title": "Case Study on Z area of Company X, Y",
    "company_a_name": "Company X",
    "company_a_summary": "Company X was founded in ... by ... It was ...",
    "company_b_name": "Company B",
    "company_b_summary": "Company B was founded in ... by ... It was ...",
    "context": [
      "Company X, a leading player in the Y industry, faced a significant challenge in Z area.",
      "This problem was critical because..."
    ],
    "approach": [
      "Step 1: Detailed description of the first key action.",
      "Step 2: Explanation of the second action, including any tools or strategies used.",
      "Step 3: Further steps segmented by tangible outputs."
    ],
    "impact": [
      "Outcome 1: Specific figures showing improvement.",
      "Outcome 2: Quantitative metrics demonstrating success.",
      "Outcome 3: Broader impact on the organization, supported by data."
    ]
  }
}
`

type Response struct {
	CaseStudy CaseStudy `json:"case_study"`
}
type CaseStudy struct {
	Title           string   `json:"title"`
	CompanyAName    string   `json:"company_a_name"`
	CompanyASummary string   `json:"company_a_summary"`
	CompanyBName    string   `json:"company_b_name"`
	CompanyBSummary string   `json:"company_b_summary"`
	Context         []string `json:"context"`
	Approach        []string `json:"approach"`
	Impact          []string `json:"impact"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func GetSummary(prompt string) (CaseStudy, error) {
	if len(prompt) < 8 {
		return CaseStudy{}, fmt.Errorf("Prompt is too short, please provide more information")
	}
	client := openai.NewClient(openaiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: SYSTEM_PROMPT,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return CaseStudy{}, fmt.Errorf("Failed to connect to AI model")
	}

	fmt.Println(resp.Choices[0].Message.Content)

	// Serialise the response
	var caseStudy Response
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &caseStudy)
	if err != nil {
		var errorResponse ErrorResponse
		err2 := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &errorResponse)
		if err2 != nil {
			fmt.Println(err, err2)
			return CaseStudy{}, fmt.Errorf("Invalid AI response, try again?")
		}
		return CaseStudy{}, fmt.Errorf(errorResponse.Error)
	}

	return caseStudy.CaseStudy, nil
}

func escape(s string) string {
	// TODO?

	return s
}

func generateSlides(caseStudy CaseStudy) (string, error) {
	data, _ := slideshow.ReadFile("page.html")
	tex := string(data)
	tex = strings.ReplaceAll(tex, "@@TITLE@@", escape(caseStudy.Title))
	tex = strings.ReplaceAll(tex, "@@AUTHOR@@", "AI")
	tex = strings.ReplaceAll(tex, "@@COMPANYA@@", escape(caseStudy.CompanyAName))
	tex = strings.ReplaceAll(tex, "@@COMPANYADESC@@", escape(caseStudy.CompanyASummary))
	tex = strings.ReplaceAll(tex, "@@COMPANYB@@", escape(caseStudy.CompanyBName))
	tex = strings.ReplaceAll(tex, "@@COMPANYBDESC@@", escape(caseStudy.CompanyBSummary))

	context := ""
	for _, c := range caseStudy.Context {
		context += "<li>" + escape(c) + "</li>\n"
	}
	tex = strings.ReplaceAll(tex, "@@CONTEXT@@", context)

	approach := ""
	for _, a := range caseStudy.Approach {
		approach += "<li>" + escape(a) + "</li>\n"
	}
	tex = strings.ReplaceAll(tex, "@@APPROACH@@", approach)

	impact := ""
	for _, i := range caseStudy.Impact {
		impact += "<li>" + escape(i) + "</li>\n"
	}
	tex = strings.ReplaceAll(tex, "@@IMPACT@@", impact)

	// Write to temp directory
	dir, err := os.MkdirTemp("", "slides")
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	doc, _ := slideshow.ReadFile("doc.svg")
	target, _ := slideshow.ReadFile("target.svg")
	arrow, _ := slideshow.ReadFile("arrow.png")
	os.WriteFile(dir+"/doc.svg", doc, 0600)
	os.WriteFile(dir+"/target.svg", target, 0600)
	os.WriteFile(dir+"/arrow.png", arrow, 0600)
	os.WriteFile(dir+"/slides.html", []byte(tex), 0600)
	// Run chromium --headless=new --print-to-pdf --window-size=1920,1080 --no-pdf-header-footer page.html
	cmd := exec.Command("chromium", "--headless=new", "--print-to-pdf", "--window-size=1920,1080", "--no-pdf-header-footer", "slides.html")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return dir + "/output.pdf", nil
}
