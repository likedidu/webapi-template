package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
)

var whitelist = map[string]bool{
	"raw.githubusercontent.com":  true,
	"gist.githubusercontent.com": true,
}

var functions = template.FuncMap{
	"encodeUrl": func(text string) string {
		return url.QueryEscape(text)
	},
	"decodeUrl": func(text string) string {
		raw, err := url.QueryUnescape(text)
		if err != nil {
			return ""
		}
		return raw
	},
	"encodeBase64": func(text string) string {
		return base64.StdEncoding.EncodeToString([]byte(text))
	},
	"decodeBase64": func(text string) string {
		raw, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return ""
		}
		return string(raw)
	},
	"encodeBase64Url": func(text string) string {
		return base64.URLEncoding.EncodeToString([]byte(text))
	},
	"decodeBase64Url": func(text string) string {
		raw, err := base64.URLEncoding.DecodeString(text)
		if err != nil {
			return ""
		}
		return string(raw)
	},
	"schemeOf": func(text string) string {
		u, err := url.Parse(text)
		if err != nil {
			return ""
		}

		return u.Scheme
	},
	"hostOf": func(text string) string {
		u, err := url.Parse(text)
		if err != nil {
			return ""
		}

		return u.Host
	},
	"pathOf": func(text string) string {
		u, err := url.Parse(text)
		if err != nil {
			return ""
		}

		return u.Path
	},
	"queriesOf": func(text string) string {
		u, err := url.Parse(text)
		if err != nil {
			return ""
		}

		return u.RawQuery
	},
}

func main() {
	if len(os.Args) < 2 {
		println("Usage: <listen-address>")

		os.Exit(1)
	}

	parser := template.New("template").Funcs(functions)

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		queries := request.URL.Query()

		var templateText string

		switch request.Method {
		case http.MethodGet:
			u := queries.Get("template")
			if u == "" {
				http.Error(writer, "Query parameter 'template' not found", http.StatusBadRequest)

				return
			}

			templateUrl, err := url.Parse(u)
			if err != nil {
				http.Error(writer, "Parse url "+queries.Get("template")+": "+err.Error(), http.StatusBadRequest)

				return
			}

			if templateUrl.Scheme != "http" && templateUrl.Scheme != "https" {
				http.Error(writer, "Unsupported url "+templateUrl.String(), http.StatusBadRequest)

				return
			}

			if !whitelist[templateUrl.Host] {
				http.Error(writer, "Template "+templateUrl.String()+" unavailable", http.StatusForbidden)

				return
			}

			resp, err := http.Get(templateUrl.String())
			if err != nil {
				http.Error(writer, "Fetch template from "+templateUrl.String()+": "+err.Error(), http.StatusForbidden)

				return
			}

			tmpl, err := io.ReadAll(resp.Body)
			if err != nil {
				http.Error(writer, "Fetch template from "+templateUrl.String()+": "+err.Error(), http.StatusForbidden)

				return
			}

			templateText = string(tmpl)
		case http.MethodPost:
			tmpl, err := io.ReadAll(request.Body)
			if err != nil {
				http.Error(writer, "Fetch request payload: "+err.Error(), http.StatusBadRequest)

				return
			}

			if len(tmpl) == 0 {
				http.Error(writer, "Empty template from body", http.StatusBadRequest)

				return
			}

			templateText = string(tmpl)
		default:
			http.Error(writer, "", http.StatusMethodNotAllowed)

			return
		}

		executor, err := parser.Parse(templateText)
		if err != nil {
			http.Error(writer, "Parse template: "+err.Error(), http.StatusForbidden)

			return
		}

		values := map[string]string{}

		for k, v := range queries {
			if len(v) == 0 {
				continue
			}

			values[k] = v[len(v)-1]
		}

		rendered := bytes.NewBuffer(nil)

		err = executor.Execute(rendered, values)
		if err != nil {
			http.Error(writer, "Render template: "+err.Error(), http.StatusBadRequest)

			return
		}

		attributes := map[string]string{}

		lines := bytes.Split(rendered.Bytes(), []byte{'\n'})
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("# Attribute: ")) {
				kv := bytes.SplitN(line[13:], []byte("="), 2)
				if len(kv) != 2 {
					continue
				}

				attributes[strings.TrimSpace(string(kv[0]))] = strings.TrimSpace(string(kv[1]))
			} else {
				break
			}
		}

		userInfo, ok := attributes["userinfo-url"]
		if ok {
			resp, err := http.Head(userInfo)
			if err == nil {
				info := resp.Header.Get("Subscription-Userinfo")
				if info != "" {
					writer.Header().Add("Subscription-Userinfo", info)
				}
			}
		}

		filename, ok := attributes["filename"]
		if ok {
			mediaType := mime.FormatMediaType("attachment", map[string]string{"filename": filename})

			writer.Header().Add("Content-Disposition", mediaType)
		}

		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(200)

		_, err = io.Copy(writer, rendered)
		if err != nil {
			return
		}
	})

	err := http.ListenAndServe(os.Args[1], handler)
	if err != nil {
		println("Listen http: " + err.Error())

		os.Exit(1)
	}
}