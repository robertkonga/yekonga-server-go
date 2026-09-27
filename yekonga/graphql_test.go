package yekonga

import (
	"context"
	"testing"

	"github.com/robertkonga/yekonga-server-go/plugins/graphql"
)

// A relation whose parent row has no link value must match nothing, even for
// resolvers that ignore setModelParams' result and run the query anyway.
func TestSetModelParamsUnlinkedRelationMatchesNothing(t *testing.T) {
	g := &GraphqlAutoBuild{}
	model := &DataModelQuery{Model: &DataModel{Name: "Tag"}}

	p := graphql.ResolveParams{
		Source:  map[string]interface{}{"name": "a post without tags"},
		Args:    map[string]interface{}{},
		Context: context.Background(),
	}

	if g.setModelParams(model, &p, "tagId", "id", false) {
		t.Fatal("setModelParams reported a link for a parent without one")
	}

	value, ok := model.where["_id"]
	if !ok || value != nil {
		t.Fatalf("where = %v, want a match-nothing _id filter", model.where)
	}
}
