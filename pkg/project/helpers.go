package project

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/shalb/cluster.dev/internal/config"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// CreateMarker generate hash string for template markers.
func (p *Project) CreateMarker(markerType string) string {
	const markerLen = 10
	hash := randSeq(markerLen)
	return fmt.Sprintf("%s.%s.%s", hash, markerType, hash)
}

func printVersion() string {
	return config.Global.Version
}

func removeDirContent(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func findModule(module Module, modsList map[string]Module) *Module {
	mod, exists := modsList[fmt.Sprintf("%s.%s", module.InfraName(), module.Name())]
	// log.Printf("Check Mod: %s, exists: %v, list %v", name, exists, modsList)
	if !exists {
		return nil
	}
	return &mod
}

// ScanMarkers use marker scanner function to replace templated markers.
func ScanMarkers(data interface{}, procFunc MarkerScanner, module Module) error {
	out := reflect.ValueOf(data)
	if out.Kind() == reflect.Ptr && !out.IsNil() {
		out = out.Elem()
	}
	switch out.Kind() {
	case reflect.Slice:
		for i := 0; i < out.Len(); i++ {
			if out.Index(i).Elem().Kind() == reflect.String {
				val, err := procFunc(out.Index(i), module)
				if err != nil {
					return err
				}
				out.Index(i).Set(val)
			} else {
				err := ScanMarkers(out.Index(i).Interface(), procFunc, module)
				if err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		for _, key := range out.MapKeys() {
			if out.MapIndex(key).Elem().Kind() == reflect.String {
				val, err := procFunc(out.MapIndex(key), module)
				if err != nil {
					return err
				}
				out.SetMapIndex(key, val)
			} else {
				err := ScanMarkers(out.MapIndex(key).Interface(), procFunc, module)
				if err != nil {
					return err
				}
			}
		}
	default:

	}
	return nil
}

func checkDependenciesRecursive(mod Module, maxDepth int) bool {
	if maxDepth == 0 {
		return false
	}
	// log.Debugf("Mod: %v, depth: %v\n%+v", mod.Name, maxDepth, mod.Dependencies)
	for _, dep := range mod.Dependencies() {
		if ok := checkDependenciesRecursive(dep.Module, maxDepth-1); !ok {
			return false
		}
	}
	return true
}

func ConvertToTfVarName(name string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err.Error())
	}
	processedString := reg.ReplaceAllString(name, "_")
	return strings.ToLower(processedString)
}

func ConvertToShellVarName(name string) string {
	return strings.ToUpper(ConvertToTfVarName(name))
}

func ConvertToShellVar(name string) string {
	return fmt.Sprintf("${%s}", ConvertToShellVarName(name))
}

func BuildDep(m Module, dep *Dependency) error {
	if dep.Module == nil {

		if dep.ModuleName == "" || dep.InfraName == "" {
			return fmt.Errorf("Empty dependency in module '%v.%v'", m.InfraName(), m.Name())
		}
		depMod, exists := m.ProjectPtr().Modules[fmt.Sprintf("%v.%v", dep.InfraName, dep.ModuleName)]
		if !exists {
			return fmt.Errorf("Error in module '%v.%v' dependency, target '%v.%v' does not exist", m.InfraName(), m.Name(), dep.InfraName, dep.ModuleName)
		}
		dep.Module = depMod
		log.Debugf("DEPENDENCY DONE! %+v", *dep)
	}
	return nil
}
