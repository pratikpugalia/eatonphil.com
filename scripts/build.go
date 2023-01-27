package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"bufio"
	"bytes"
	"text/template"
)

func copyFile(in, out string) {
	r, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	w, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	// TODO: validate complete read/write
	_, err = w.ReadFrom(r)
	if err != nil {
		panic(err)
	}
}

var SECTIONS = []struct{
	name string
	ctx map[string]any
}{
	{
		"notes",
		map[string]any{
			"Tag": "Notes by a software developer",
			"Section": "Notes",
		},
	},
	{
		"letters",
		map[string]any{
			"Tag": "Letters by a software developer",
			"Section": "Letters",
		},
	},
	{
		"lists",
		map[string]any{
			"Tag": "Phil Eaton's Lists",
			"Section": "Lists",
		},
	},
	{
		"home",
		map[string]any{
			"Tag": "Phil Eaton",
			"Section": "",
		},
	},
}

type Doc struct {
	Title string
	Date string
	Tags []string
	Body string
	CanonicalURL string
}

func parseDoc(docPath string) Doc {
	contents, err := os.ReadFile(docPath)
	if err != nil {
		panic(err)
	}

	sections := strings.Split(string(contents), "---")
	header := sections[0]
	body := sections[1]

	var d Doc

	for _, line := range strings.Split(header, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "=")
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "title":
			d.Title = value
		case "date":
			d.Date = value
		case "tags":
			d.Tags = strings.Split(value, ",")
		}
	}

	var out bytes.Buffer
	outWriter := bufio.NewWriter(&out)
	bodyRunes := []rune(body)
	var prev rune
	var i int
	for i < len(bodyRunes) {
		if i > 0 {
			prev = bodyRunes[i-1]
		}

		c := bodyRunes[i]
		// Deal with headers
		if (i == 0 || prev == '\n') && c == '#' {
			var headerNumber = 0
			for c == '#' {
				i++
				headerNumber++
				c = bodyRunes[i]
			}

			var header []rune
			for !(c == '\r' || c == '\n') {
				i++
				header = append(header, c)
				c = bodyRunes[i]
			}

			fmt.Fprintf(outWriter, "<h%d>%s</h%d>", headerNumber, string(header), headerNumber)
			continue
		}

		outWriter.WriteRune(c)
		i++
	}
	outWriter.Flush()
	d.Body = string(out.Bytes())

	return d
}

func buildSection(t *template.Template, section string, ctx map[string]any) {
	files, err := os.ReadDir(path.Join(section, "posts"))
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		inputPath := path.Join(section, "posts", f.Name())
		fmt.Println("Building", inputPath)
		doc := parseDoc(inputPath)

		outputPath := path.Join(section, "build", f.Name())
		if strings.HasSuffix(f.Name(), ".md") {
			outputPath = outputPath[:len(outputPath)-4] + ".html"
		}

		o, err := os.Create(outputPath)
		if err != nil  {
			panic(err)
		}

		canonicalEnd := "/"
		if f.Name() != "index.html" {
			canonicalEnd = "/" + f.Name()
		}
		doc.CanonicalURL = ctx["Domain"].(string) + canonicalEnd
				
		ctx["Page"] = doc
		err = t.Execute(o, ctx)
		if err != nil {
			panic(err)
		}

		err = o.Close()
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	mailFile, err := os.ReadFile("mail.html")
	if err != nil {
		panic(err)
	}

	templateFile, err := os.ReadFile("template.html")
	if err != nil {
		panic(err)
	}

	t, err := template.New("template").Parse(string(templateFile))
	if err != nil {
		panic(err)
	}
	for _, section := range SECTIONS {
		fmt.Println("Building section", section.name)
		if section.name == "notes" {
			continue
		}

		if _, err := os.Stat(section.name); err != nil {
			if os.IsNotExist(err) {
				panic("Section does not exist:" + section.name)
			}
		}

		// TODO: validate the section exists?
		buildPath := path.Join(section.name, "build")
		err := os.RemoveAll(buildPath)
		if err != nil {
			panic(err)
		}

		err = os.MkdirAll(buildPath, os.ModePerm)
		if err != nil {
			panic(err)
		}

		// Copy in style.css
		copyFile("style.css", path.Join(buildPath, "style.css"))

		// Render all templates
		section.ctx["Mail"] = string(mailFile)
		section.ctx["Domain"] = "eatonphil.com"
		if section.name != "home" {
			section.ctx["Domain"] = section.name + ".eatonphil.com"
		}
		buildSection(t, section.name, section.ctx)
	}
}