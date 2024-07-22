package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flamego/binding"
	"github.com/flamego/flamego"
	"github.com/flamego/session"
	"github.com/flamego/template"
	"github.com/google/uuid"

	"github.com/humaidq/case-study-gen/static"
	"github.com/humaidq/case-study-gen/templates"
)

type AIForm struct {
	Prompt string `form:"prompt"`
}

//go:embed page.html
//go:embed *.svg
//go:embed *.png
var slideshow embed.FS

type Flash struct {
	Success string
	Error   string
}

type StudyStatus int

const (
	StudyStatusPending StudyStatus = iota
	StudyStatusComplete
	StudyStatusFailed
)

type StudyRequest struct {
	study  *CaseStudy
	status StudyStatus
	file   string
	err    error
}

var studies map[string]*StudyRequest

func ProcessPrompt(id string, prompt string) {
	study, err := GetSummary(prompt)
	if err != nil {
		studies[id].err = err
		studies[id].status = StudyStatusFailed
		return
	}

	fmt.Println(study)
	output, err := generateSlides(study)
	if err != nil {
		studies[id].err = fmt.Errorf("Failed to generate slides")
		studies[id].status = StudyStatusFailed
		return
	}

	studies[id].study = &study
	studies[id].status = StudyStatusComplete
	studies[id].file = output
}

var (
	port      string
	openaiKey string
)

func loadEnvs() {
	port = os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	keyPath := os.Getenv("OPENAI_KEY_PATH")
	if len(keyPath) == 0 {
		openaiKey = os.Getenv("OPENAI_KEY")
	} else {
		b, err := os.ReadFile(keyPath)
		if err != nil {
			log.Fatalf("Failed to read OpenAI key file: %v", err)
		}
		openaiKey = strings.TrimSpace(string(b))
	}

	if len(openaiKey) == 0 {
		log.Fatalf("OpenAI key not set!")
	}
}

func main() {
	studies = make(map[string]*StudyRequest)
	loadEnvs()

	f := flamego.Classic()

	// Setup flamego
	fs, err := template.EmbedFS(templates.Templates, ".", []string{".html"})
	if err != nil {
		panic(err)
	}
	f.Use(session.Sessioner())
	f.Use(template.Templater(template.Options{
		FileSystem: fs,
	}))
	f.Use(flamego.Static(flamego.StaticOptions{
		FileSystem: http.FS(static.Static),
	}))

	// Routes
	f.Get("/", func(c flamego.Context, t template.Template) {
		t.HTML(http.StatusOK, "home")
	})

	f.Post("/", binding.Form(AIForm{}), func(c flamego.Context,
		data template.Data, form AIForm, errs binding.Errors,
		t template.Template) {

		id := uuid.New().String()
		studies[id] = &StudyRequest{
			status: StudyStatusPending,
		}

		go ProcessPrompt(id, form.Prompt)
		c.Redirect("/study/" + id)
		t.HTML(http.StatusOK, "loading")
	})

	f.Get("/study/{id}.pdf", func(c flamego.Context, s session.Session) []byte {
		id := c.Param("id")
		study, ok := studies[id]
		if !ok {
			fmt.Println(err)
			return []byte{}
		}

		b, err := os.ReadFile(study.file)
		if err != nil {
			fmt.Println(err)
			return []byte{}
		}

		return b
	})

	f.Get("/study/{id}/status", func(c flamego.Context) (int, string) {
		id := c.Param("id")
		status, ok := studies[id]
		if !ok {
			return http.StatusOK, "Not found"
		}
		if status.status == StudyStatusComplete {
			return http.StatusOK, "OK"
		}
		return http.StatusOK, "Not ready"
	})

	f.Get("/study/{id}", func(c flamego.Context, t template.Template,
		d template.Data, s session.Session) {
		id := c.Param("id")
		d["id"] = id
		status, ok := studies[id]
		if !ok {
			s.SetFlash(Flash{
				Error: "Study not found",
			})
			c.Redirect("/")
			return
		}

		switch status.status {
		case StudyStatusPending:
			t.HTML(http.StatusOK, "loading")
			return
		case StudyStatusComplete:
			t.HTML(http.StatusOK, "summary")
			return
		}

		s.SetFlash(Flash{
			Error: "Study not found",
		})
		c.Redirect("/")

	})

	// Serve
	log.Printf("Starting web server on port %s\n", port)
	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
		Handler:      f,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
