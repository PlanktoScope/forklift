package forklift

import (
	"fmt"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/pkg/errors"

	fplt "github.com/forklift-run/forklift/pkg/pallets"
	"github.com/forklift-run/forklift/pkg/structures"
)

const (
	AddReconciliationChange    = "add"
	RemoveReconciliationChange = "remove"
	UpdateReconciliationChange = "update"
)

type ReconciliationChange struct {
	Name string
	Type string
	Depl *fplt.ResolvedDepl // this is nil for an app to be removed
	App  api.Stack          // this is empty for an app which does not yet exist
}

func (c *ReconciliationChange) String() string {
	if c.Depl == nil {
		return fmt.Sprintf("(%s %s)", c.Type, c.Name)
	}
	return fmt.Sprintf("(%s %s)", c.Type, c.Depl.Name)
}

func (c *ReconciliationChange) PlanString() string {
	if c.Depl == nil {
		return fmt.Sprintf("%s Compose app %s (from unknown deployment)", c.Type, c.Name)
	}
	return fmt.Sprintf("%s deployment %s as Compose app %s", c.Type, c.Depl.Name, c.Name)
}

func NewAddReconciliationChange(
	deplName string, depl *fplt.ResolvedDepl,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: fplt.GetComposeAppName(deplName),
		Type: AddReconciliationChange,
		Depl: depl,
	}
}

func NewUpdateReconciliationChange(
	deplName string, depl *fplt.ResolvedDepl, app api.Stack,
) *ReconciliationChange {
	return &ReconciliationChange{
		Name: fplt.GetComposeAppName(deplName),
		Type: UpdateReconciliationChange,
		Depl: depl,
		App:  app,
	}
}

func NewRemoveReconciliationChange(appName string, app api.Stack) *ReconciliationChange {
	return &ReconciliationChange{
		Name: appName,
		Type: RemoveReconciliationChange,
		App:  app,
	}
}

// identifyReconciliationChanges builds an arbitrarily-ordered list of changes to carry out to
// reconcile the desired list of deployments with the actual list of active Docker Compose apps.
func identifyReconciliationChanges(
	depls []*fplt.ResolvedDepl, apps []api.Stack,
) ([]*ReconciliationChange, error) {
	deplsByName := make(map[string]*fplt.ResolvedDepl)
	for _, depl := range depls {
		deplsByName[depl.Name] = depl
	}
	appsByName := make(map[string]api.Stack)
	for _, app := range apps {
		appsByName[app.Name] = app
	}
	composeAppDefinerSet, err := identifyComposeAppDefiners(deplsByName)
	if err != nil {
		return nil, err
	}

	appDeplNames := make(map[string]string)
	changes := make([]*ReconciliationChange, 0, len(depls)+len(apps))
	for name, depl := range deplsByName {
		appDeplNames[fplt.GetComposeAppName(name)] = name
		app, ok := appsByName[fplt.GetComposeAppName(name)]
		if !ok {
			if composeAppDefinerSet.Has(name) {
				changes = append(changes, NewAddReconciliationChange(name, depl))
			}
			continue
		}
		if composeAppDefinerSet.Has(name) {
			changes = append(changes, NewUpdateReconciliationChange(name, depl, app))
		}
	}
	for name, app := range appsByName {
		if deplName, ok := appDeplNames[name]; ok {
			if composeAppDefinerSet.Has(deplName) {
				continue
			}
		}
		changes = append(changes, NewRemoveReconciliationChange(name, app))
	}
	return changes, nil
}

// identifyComposeAppDefiners builds a set of the names of deployments which define Compose apps.
func identifyComposeAppDefiners(
	depls map[string]*fplt.ResolvedDepl,
) (structures.Set[string], error) {
	composeAppDefinerSet := make(structures.Set[string])
	for _, depl := range depls {
		definesApp, err := depl.DefinesComposeApp()
		if err != nil {
			return nil, errors.Wrapf(
				err, "couldn't determine whether package deployment %s defines a Compose app", depl.Name,
			)
		}
		if definesApp {
			composeAppDefinerSet.Add(depl.Name)
		}
	}
	return composeAppDefinerSet, nil
}
