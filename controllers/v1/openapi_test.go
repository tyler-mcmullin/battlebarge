package v1_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"

	"battlebarge/routes"
	"battlebarge/testutil"
)

// These tests keep docs/openapi.yaml in step with the code: the documented
// routes must be exactly the registered ones, and real responses must have
// exactly the fields their schemas list.

func loadSpec(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec map[string]any
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return spec
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func TestOpenAPIMatchesRoutes(t *testing.T) {
	spec := loadSpec(t)

	documented := map[string]bool{}
	for path, item := range asMap(spec["paths"]) {
		for method := range asMap(item) {
			if method == "parameters" {
				continue
			}
			// {id} in OpenAPI is :id in gin
			p := path
			for _, seg := range strings.Split(path, "/") {
				if strings.HasPrefix(seg, "{") {
					p = strings.Replace(p, seg, ":"+strings.Trim(seg, "{}"), 1)
				}
			}
			documented[strings.ToUpper(method)+" "+p] = true
		}
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.GetAuthControllers(r)
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)
	routes.GetUnitControllers(r)
	routes.GetCampaignControllers(r)

	registered := map[string]bool{}
	for _, ri := range r.Routes() {
		registered[ri.Method+" "+ri.Path] = true
	}

	for route := range registered {
		if !documented[route] {
			t.Errorf("route %s is not in docs/openapi.yaml", route)
		}
	}
	for route := range documented {
		if !registered[route] {
			t.Errorf("docs/openapi.yaml documents %s, which is not a registered route", route)
		}
	}
}

// Every documented operation must declare the responses the handlers give for
// its own protection level: operations with bearerAuth must document 401.
func TestOpenAPISecurityMatchesRoutes(t *testing.T) {
	spec := loadSpec(t)

	for path, item := range asMap(spec["paths"]) {
		for method, op := range asMap(item) {
			if method == "parameters" {
				continue
			}
			o := asMap(op)
			_, secured := o["security"]
			responses := asMap(o["responses"])
			if _, has401 := responses["401"]; secured && !has401 {
				t.Errorf("%s %s requires auth but does not document 401", strings.ToUpper(method), path)
			}
			// authenticated routes also reject unverified emails with 403
			if _, has403 := responses["403"]; secured && !has403 {
				t.Errorf("%s %s requires auth but does not document 403 (email not verified)", strings.ToUpper(method), path)
			}
		}
	}
}

// resolve follows a {$ref: '#/components/schemas/X'} to the schema it names.
func resolve(spec map[string]any, schema map[string]any) map[string]any {
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	return asMap(asMap(asMap(spec["components"])["schemas"])[name])
}

// checkShape verifies value (decoded JSON) has exactly the fields the schema
// lists, recursing into objects and arrays, and that types are plausible.
func checkShape(t *testing.T, spec map[string]any, schema map[string]any, value any, path string) {
	t.Helper()
	schema = resolve(spec, schema)

	switch schema["type"] {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			t.Errorf("%s: want object, got %T", path, value)
			return
		}
		props := asMap(schema["properties"])
		for key := range obj {
			if _, ok := props[key]; !ok {
				t.Errorf("%s.%s is returned but not in the schema", path, key)
			}
		}
		for key, p := range props {
			v, ok := obj[key]
			if !ok {
				// fields marked x-owner-only are deliberately left out of responses
				// the owner is not the audience for
				if asMap(p)["x-owner-only"] == true {
					continue
				}
				t.Errorf("%s.%s is in the schema but not returned", path, key)
				continue
			}
			checkShape(t, spec, asMap(p), v, path+"."+key)
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			t.Errorf("%s: want array, got %T (null arrays break clients)", path, value)
			return
		}
		for _, el := range arr {
			checkShape(t, spec, asMap(schema["items"]), el, path+"[]")
		}
	case "string":
		if _, ok := value.(string); !ok {
			t.Errorf("%s: want string, got %T", path, value)
		}
	case "integer":
		if f, ok := value.(float64); !ok || f != float64(int64(f)) {
			t.Errorf("%s: want integer, got %v", path, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			t.Errorf("%s: want boolean, got %T", path, value)
		}
	}
}

func TestOpenAPIResponseShapes(t *testing.T) {
	testutil.SetupDB(t)
	spec := loadSpec(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	check := func(schemaName string, w interface{ Bytes() []byte }, label string) {
		t.Helper()
		var v any
		if err := json.Unmarshal(w.Bytes(), &v); err != nil {
			t.Fatalf("%s: decode: %v", label, err)
		}
		schema := asMap(asMap(asMap(spec["components"])["schemas"])[schemaName])
		checkShape(t, spec, schema, v, label)
	}
	body := func(method, path, payload string, want int) *bodyBytes {
		t.Helper()
		uid := "owner"
		w := call(r, uid, method, path, payload)
		expect(t, w, want)
		return &bodyBytes{w.Body.Bytes()}
	}

	// warband: empty at creation (units must be [] not null), then with a unit and a perk
	wb := body("POST", "/warbands/create", `{"name":"Da Boyz"}`, http.StatusCreated)
	check("Warband", wb, "create warband")
	var created struct{ ID string }
	json.Unmarshal(wb.Bytes(), &created)

	unit := body("POST", "/units/create", `{"warband_id":"`+created.ID+`","unit_name":"Boss"}`, http.StatusCreated)
	check("Unit", unit, "create unit")
	var u struct{ ID string }
	json.Unmarshal(unit.Bytes(), &u)
	check("Unit", body("PATCH", "/units/"+u.ID+"/perk", `{"name":"Tough","description":"d"}`, http.StatusOK), "add perk")
	check("Warband", body("GET", "/warbands/"+created.ID, ``, http.StatusOK), "get warband")
	listed := body("GET", "/warbands", ``, http.StatusOK)
	var arr []any
	json.Unmarshal(listed.Bytes(), &arr)
	checkShape(t, spec, map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Warband"}}, any(arr), "list warbands")

	// campaign with a chapter, a team, and a member warband
	camp := body("POST", "/campaigns/create", `{"name":"Crusade"}`, http.StatusCreated)
	check("Campaign", camp, "create campaign")
	var c struct{ ID string }
	json.Unmarshal(camp.Bytes(), &c)
	cp := "/campaigns/" + c.ID
	check("CampaignChapter", body("POST", cp+"/chapters", `{"title":"One"}`, http.StatusCreated), "create chapter")
	team := body("POST", cp+"/teams", `{"name":"Red"}`, http.StatusCreated)
	check("CampaignTeam", team, "create team")
	var tm struct{ ID string }
	json.Unmarshal(team.Bytes(), &tm)
	check("Campaign", body("POST", cp+"/warbands", `{"warband_id":"`+created.ID+`","team_id":"`+tm.ID+`"}`, http.StatusCreated), "join campaign")
	check("Campaign", body("GET", cp, ``, http.StatusOK), "get campaign")

	// users/me has a different path through middleware, so check the model directly
	userJSON, _ := json.Marshal(testutil.InsertUser(t, "shape"))
	check("User", &bodyBytes{userJSON}, "user")

	// error bodies
	check("Error", body("GET", "/warbands/"+uuid.NewString(), ``, http.StatusNotFound), "404 body")

	// every documented request schema's required fields must actually be required:
	// sending an empty object is a 400
	for _, p := range []string{"/warbands/create", "/units/create", "/campaigns/create", cp + "/chapters", cp + "/teams", cp + "/warbands"} {
		expect(t, call(r, "owner", "POST", p, `{}`), http.StatusBadRequest)
	}
}

type bodyBytes struct{ b []byte }

func (b *bodyBytes) Bytes() []byte { return b.b }
