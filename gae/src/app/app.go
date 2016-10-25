package demo

import (
	"math/rand"
	"net/http"
	"regexp"
	"text/template"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

var configTemplate *template.Template
var originExp *regexp.Regexp

const configString = `{
    "requests": {
        "pageview": "https://demo.ymotongpoo.com/analytics?uuid=${uuid}&url=${host}&origin=${origin}"
    },
    "vars": {
        "uuid": "{{.UUID}}",
        "host": "{{.URL}}",
        "origin": "{{.Origin}}"
    },
    "triggers": {
        "trackPageview": {
            "on": "visible",
            "request": "pageview"
        }
    }
}`

const (
	AMPCacheDomain  = "cdn.ampproject.org"
	AMPCacheOrigin  = `https://` + AMPCacheDomain
	PublisherDomain = `demo.ymotongpoo.com`
	PublisherOrigin = `https://` + PublisherDomain
    PublisherOriginNonSSL = `http://` + PublisherDomain // in the case of Google AMP viewer
)

func init() {
	// routers
	http.HandleFunc("/config.json", handleConfig)
	http.HandleFunc("/analytics", handleAnalytics)

	// init utilities
	configTemplate = template.Must(template.New("config").Parse(configString))
}

// AnalyticsData is a struct to hold data handed to config.json template.
type AnalyticsData struct {
	UUID   string
	URL    string
	Origin string
}

// NewAnalyticsData returns AnalyticsData with uuid and URL.
func NewAnalyticsData(uuid, url, origin string) AnalyticsData {
	return AnalyticsData{
		UUID:   uuid,
		URL:    url,
		Origin: origin,
	}
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	validOrigin, ok := verifyOrigin(r)
	log.Infof(ctx, "valid origin: %v", validOrigin)
	if !ok {
		http.Error(w, "Invalid Origin", http.StatusBadRequest)
		return
	}

	var data AnalyticsData
	currentCookie, err := r.Cookie("uuid")
	if err != nil {
		data = NewAnalyticsData("error", "error", getOrigin(r))
	}
	if currentCookie == nil {
		cookie := http.Cookie{
			Name:  "uuid",
			Value: randID(),
		}
		http.SetCookie(w, &cookie)
		data = NewAnalyticsData(cookie.Value, r.URL.String(), getOrigin(r))
	} else {
		data = NewAnalyticsData(currentCookie.Value, r.URL.String(), getOrigin(r))
	}

	w.Header().Add("Access-Control-Allow-Credentials", "true")
    w.Header().Add("Access-Control-Allow-Origin", AMPCacheOrigin)
	w.Header().Add("AMP-Access-Control-Allow-Source-Origin", validOrigin)
	w.Header().Add("Access-Control-Expose-Headers", "AMP-Access-Control-Allow-Source-Origin")
	configTemplate.Execute(w, data)
}

func handleAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	query := r.URL.Query()
	log.Infof(ctx, "uuid: %s, url: %s, origin: %s", query.Get("uuid"), query.Get("url"), query.Get("origin"))
}

func verifyOrigin(r *http.Request) (string, bool) {
	ctx := appengine.NewContext(r)
	origin := r.Header.Get("Origin")
	if len(origin) > 0 {
		log.Infof(ctx, "origin: %s", origin)
		query := r.URL.Query().Get("__amp_source_origin")
        log.Infof(ctx, "query: %s", query)
        switch query {
        case PublisherOrigin:
            return PublisherOrigin, true
        case PublisherOriginNonSSL:
            return PublisherOriginNonSSL, true
        case AMPCacheOrigin:
            return AMPCacheOrigin, true
        default:
            return "", false
        }
	}
	isAMPSameOrigin := r.Header.Get("AMP-Same-Origin")
	if isAMPSameOrigin == "true" {
		return PublisherOrigin, true
	}
	return "", false
}

func getOrigin(r *http.Request) string {
    origin := r.Header.Get("Origin")
    if len(origin) != 0 {
        return origin
    }
    return r.Referer()
}

/* Helpers */
const (
	letters  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	idLength = 16
)

func randID() string {
	b := make([]byte, idLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
