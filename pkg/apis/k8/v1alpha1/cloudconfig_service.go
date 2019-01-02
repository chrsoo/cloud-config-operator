package v1alpha1

import (
    "reflect"
    "encoding/json"
)

// ConfigService manages apps in a given environment 
type ConfigService interface {
    // Reconcile CloudConfig
    ReconcileConfig(cloudConfg CloudConfig)
    // Reconcile apps in the given environment
    ReconcileAppsInEnv(env Environment)
}

func getApps(env Environment) []string {

    body := env.GetAppConfig(env.AppName)
    // marshal the body as a map
    var m map[string]interface{}
    err := json.Unmarshal(body, &m)
    if err != nil {
        // TODO raise error
    }

    // make sure we return a slice even if there is a single string value
    if apps, exists := m[env.AppList]; exists {
        kind := reflect.ValueOf(apps).Kind()
        switch kind {
            case reflect.Slice:
                return apps.([]string)
            case reflect.String:
                return []string{ apps.(string) }
            default:
                // TODO raise error
                return nil
        }
    } else {
        // TODO Log warning if there is no AppsList key in the map
        return nil
    }

}

// ReconcileAppsInEnv apps in a given environement
func ReconcileAppsInEnv(env Environment) {
    // TODO ensure environment namespace
    for _, app := range getApps(env) {
        file := env.GetAppConfigFile(app, env.SpecFile)
        // TODO apply the spec file
        println(file)
    }
}