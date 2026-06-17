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

	model, cmd := app.openScopedPods(workloadSelectorMsg{
		podsRes:  podsRes,
		ns:       "prod",
		desc:     "deployments/api",
		selector: "app=api",
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

func TestToggleAllNSClearsScope(t *testing.T) {
	app := deploymentsApp()
	app.scope = podScope{selector: "app=api", desc: "deployments/api"}

	model, _ := app.toggleAllNS()
	got := model.(App)
	if got.scope.selector != "" {
		t.Fatalf("scope.selector = %q, want cleared after toggleAllNS", got.scope.selector)
	}
}
