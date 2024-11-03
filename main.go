package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)

type Response struct {
	Selector string   `json:"selector"`
	URL      string   `json:"url"`
	Elements []string `json:"elements"`
}

type CachedResponse struct {
	Timestamp time.Time
	Data      []byte
}

var (
	flagHost = flag.String("host", "localhost", "Host to listen on")
	flagPort = flag.Int("port", 8080, "Port to listen on")
	cache    = expirable.NewLRU[string, CachedResponse](10, nil, time.Hour*1)
)

func main() {
	flag.Parse()

	ln, err := reuseport.Listen("tcp4", fmt.Sprintf("%s:%d", *flagHost, *flagPort))
	if err != nil {
		log.Error().Err(err).Msg("Error opening listener")
	}

	log.Info().Str("address", ln.Addr().String()).Msg("Server started")

	if err = fasthttp.Serve(ln, handleRequest); err != nil {
		log.Error().Err(err).Msg("Error starting server")
	}
}

func handleRequest(ctx *fasthttp.RequestCtx) {
	log.Info().Str("path", string(ctx.Path())).Str("remote", ctx.RemoteIP().String()).Msg("Request received")

	if len(ctx.Path()) != 1 {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	url := formatURL(string(ctx.FormValue("url")))
	selector := string(ctx.FormValue("selector"))
	isRaw := len(ctx.FormValue("raw")) > 0
	cacheKey := url + selector + strconv.FormatBool(isRaw)

	if val, ok := cache.Get(cacheKey); ok {
		log.Info().Str("url", url).Str("selector", selector).Msg("Cache hit")

		age := time.Since(val.Timestamp.Add(time.Hour)).Abs()

		ctx.SetContentType("application/json")
		ctx.Response.Header.Set("Cache-Control", fmt.Sprintf("public, max-age=%.0f, immutable", age.Seconds()))
		ctx.SetBody(val.Data)
		return
	}

	site, err := loadSite(url)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	resp := Response{
		Selector: selector,
		URL:      url,
		Elements: []string{},
	}

	if selector != "" {
		site.Find(selector).Each(func(i int, s *goquery.Selection) {
			if isRaw {
				html, _ := s.Html()
				resp.Elements = append(resp.Elements, html)
				return
			}
			resp.Elements = append(resp.Elements, strings.TrimSpace(s.Text()))
		})
	}

	result, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	c := CachedResponse{
		Timestamp: time.Now(),
		Data:      result,
	}

	cache.Add(cacheKey, c)
	ctx.SetContentType("application/json")
	ctx.Response.Header.Set("Cache-Control", "public, max-age=3600, immutable")
	ctx.Write(result)
}

func formatURL(url string) string {
	re := regexp.MustCompile(`(?m)(http(?:s)?):\/+`)
	url = re.ReplaceAllString(url, "$1://")

	if !strings.HasPrefix(url, "http") {
		return "https://" + url
	}

	return url
}

func loadSite(url string) (*goquery.Document, error) {
	log.Info().Str("url", url).Msg("Loading site")
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	return goquery.NewDocumentFromReader(res.Body)
}
