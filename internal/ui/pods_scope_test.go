package ui

import (
	"errors"
	"testing"

	"github.com/bjarneo/ku/internal/k8s"
)

func deploymentsApp() App {
	th := PickTheme("ansi")
	app := App{
		client:    &k8s.Client{},
		theme:     th,
		keys:      defaultKeys(),
		width:     80,
		height:    20,
		screen:    screenTable,
		res:       k8s.ResourceInfo{Group: "apps", Resource: "deployments", Kind: "Deployment", Namespaced: true},
		namespace: "default",
		focus:     focusMain,
	}
	app.table = newTableView(th)
	app.table.setData(&k8s.Table{
		Columns: []k8s.Column{{Name: "Name"}},
		Rows:    []k8s.Row{{Namespace: "default", Name: "api", Cells: []string{"api"}}},
	})
	return app
}

func TestPodsKeyStartsSelectorLookup(t *testing.T) {
	app := deploymentsApp()

	model, cmd := app.updateMainKeys(mkKey("p"))
	got := model.(App)
	if cmd == nil {
		t.Fatal("pods key returned nil command")
	}
	if got.status == "" || got.statusErr {
		t.Fatalf("status = %q err=%t, want non-error loading status", got.status, got.statusErr)
	}
}

func TestPodsKeyRequiresWorkload(t *testing.T) {
	app := deploymentsApp()
	app.res = k8s.ResourceInfo{Resource: "services", Kind: "Service", Namespaced: true}

	model, cmd := app.updateMainKeys(mkKey("p"))
	got := model.(App)
	if cmd != nil {
		t.Fatal("non-workload pods key returned command")
	}
	if !got.statusErr {
		t.Fatalf("status = %q err=%t, want error status", got.status, got.statusErr)
	}
}

func TestOpenScopedPodsSwitchesAndScopes(t *testing.T) {
	app := deploymentsApp()
	podsRes := k8s.ResourceInfo{Resource: "pods", Kind: "Pod", Namespaced: true}
	origin := target{res: app.res, ns: "", name: "api"}

	model, cmd := app.openScopedPods(workloadSelectorMsg{
		podsRes:  podsRes,
		ns:       "prod",
		desc:     "deployments/api",
		selector: "app=api",
		origin:   origin,
	})
	got := model.(App)
	if cmd == nil {
		t.Fatal("openScopedPods returned nil command")
	}
	if !got.res.IsPod() {
		t.Fatalf("res = %+v, want pods", got.res)
	}
	if got.namespace != "prod" {
		t.Fatalf("namespace = %q, want prod (narrowed to the workload's namespace)", got.namespace)
	}
	if got.scope.selector != "app=api" || got.scope.desc != "deployments/api" {
		t.Fatalf("scope = %+v, want app=api / deployments/api", got.scope)
	}
	if got.scope.origin.name != "api" || !got.scope.origin.res.IsDeployment() {
		t.Fatalf("scope.origin = %+v, want the deployments/api workload", got.scope.origin)
	}
}

func TestExitScopeReturnsToWorkload(t *testing.T) {
	app := deploymentsApp()
	app.res = k8s.ResourceInfo{Resource: "pods", Kind: "Pod", Namespaced: true}
	app.namespace = "prod"
	app.scope = podScope{
		selector: "app=api",
		desc:     "deployments/api",
		origin:   target{res: k8s.ResourceInfo{Group: "apps", Resource: "deployments", Kind: "Deployment", Namespaced: true}, ns: "default", name: "api"},
	}

	model, cmd := app.exitScope()
	got := model.(App)
	if cmd == nil {
		t.Fatal("exitScope returned nil command")
	}
	if !got.res.IsDeployment() {
		t.Fatalf("res = %+v, want deployments (the origin)", got.res)
	}
	if got.namespace != "default" {
		t.Fatalf("namespace = %q, want default (the origin's list namespace)", got.namespace)
	}
	if got.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want cleared after exit", got.scope.selector)
	}
	if got.pendingSelect.name != "api" {
		t.Fatalf("pendingSelect.name = %q, want api (re-select the workload)", got.pendingSelect.name)
	}
}

func TestBackExitsScopeOnTable(t *testing.T) {
	app := deploymentsApp()
	app.res = k8s.ResourceInfo{Resource: "pods", Kind: "Pod", Namespaced: true}
	app.scope = podScope{
		selector: "app=api",
		origin:   target{res: k8s.ResourceInfo{Group: "apps", Resource: "deployments", Kind: "Deployment", Namespaced: true}, ns: "default", name: "api"},
	}

	model, _ := app.updateTable(mkKey("esc"))
	got := model.(App)
	if !got.res.IsDeployment() {
		t.Fatalf("res = %+v, want deployments after esc", got.res)
	}
	if got.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want cleared after esc", got.scope.selector)
	}
}

func TestBackClearsFilterBeforeExitingScope(t *testing.T) {
	app := deploymentsApp()
	app.res = k8s.ResourceInfo{Resource: "pods", Kind: "Pod", Namespaced: true}
	app.scope = podScope{selector: "app=api", origin: target{res: app.res, ns: "default", name: "api"}}
	app.table.startFilter()
	app.table.filter.SetValue("foo")

	model, _ := app.updateTable(mkKey("esc"))
	got := model.(App)
	if got.scope.selector == "" {
		t.Fatal("first esc exited the scope; it should clear the filter first")
	}
	if got.table.filterActive() {
		t.Fatal("first esc did not clear the active filter")
	}
}

func TestResourcesLoadedAppliesPendingSelect(t *testing.T) {
	app := deploymentsApp()
	app.loadSeq = 5
	app.pendingSelect = target{name: "api"}

	msg := resourcesLoadedMsg{
		client: app.client,
		seq:    5,
		res:    app.res,
		ns:     app.namespace,
		tbl: &k8s.Table{
			Columns: []k8s.Column{{Name: "Name"}},
			Rows: []k8s.Row{
				{Namespace: "default", Name: "web", Cells: []string{"web"}},
				{Namespace: "default", Name: "api", Cells: []string{"api"}},
			},
		},
	}
	model, _ := app.Update(msg)
	got := model.(App)
	row, ok := got.table.selected()
	if !ok || row.Name != "api" {
		t.Fatalf("selected row = %+v ok=%t, want api", row, ok)
	}
	if got.pendingSelect.name != "" {
		t.Fatalf("pendingSelect = %+v, want cleared after applying", got.pendingSelect)
	}
}

func TestOpenScopedPodsError(t *testing.T) {
	app := deploymentsApp()

	model, cmd := app.openScopedPods(workloadSelectorMsg{
		ns:   "default",
		desc: "deployments/api",
		err:  errors.New("boom"),
	})
	got := model.(App)
	if cmd != nil {
		t.Fatal("openScopedPods on error returned a command")
	}
	if got.res.IsPod() {
		t.Fatal("openScopedPods on error switched to pods")
	}
	if got.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want empty on error", got.scope.selector)
	}
	if !got.statusErr {
		t.Fatalf("status = %q err=%t, want error status", got.status, got.statusErr)
	}
}

func TestSwitchingResourceClearsScope(t *testing.T) {
	app := deploymentsApp()
	app.scope = podScope{selector: "app=api", desc: "deployments/api"}

	app.useResource(k8s.ResourceInfo{Resource: "services", Kind: "Service", Namespaced: true})
	if app.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want cleared after useResource", app.scope.selector)
	}
}

func TestPodsKeyFromConfigView(t *testing.T) {
	app := deploymentsApp()
	app.screen = screenConfig
	app.configTarget = target{
		res:  k8s.ResourceInfo{Group: "apps", Resource: "deployments", Kind: "Deployment", Namespaced: true},
		ns:   "default",
		name: "api",
	}

	_, cmd := app.updateConfig(mkKey("p"))
	if cmd == nil {
		t.Fatal("p in config view returned nil command; want a selector lookup")
	}
}

func TestPodsKeyFromDetailView(t *testing.T) {
	app := deploymentsApp()
	app.screen = screenDetail
	app.detailTarget = target{
		res:  k8s.ResourceInfo{Group: "apps", Resource: "statefulsets", Kind: "StatefulSet", Namespaced: true},
		ns:   "default",
		name: "db",
	}

	_, cmd := app.updateDetail(mkKey("p"))
	if cmd == nil {
		t.Fatal("p in detail view returned nil command; want a selector lookup")
	}
}

func TestPodsKeyFromConfigViewRequiresWorkload(t *testing.T) {
	app := deploymentsApp()
	app.screen = screenConfig
	app.configTarget = target{
		res:  k8s.ResourceInfo{Resource: "services", Kind: "Service", Namespaced: true},
		ns:   "default",
		name: "api",
	}

	model, cmd := app.updateConfig(mkKey("p"))
	got := model.(App)
	if cmd != nil {
		t.Fatal("p on a non-workload config target returned a command")
	}
	if !got.statusErr {
		t.Fatalf("status = %q err=%t, want error status", got.status, got.statusErr)
	}
}

func TestToggleAllNSClearsScope(t *testing.T) {
	app := deploymentsApp()
	app.scope = podScope{selector: "app=api", desc: "deployments/api"}

	model, _ := app.toggleAllNS()
	got := model.(App)
	if got.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want cleared after toggleAllNS", got.scope.selector)
	}
}
